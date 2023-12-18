package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"

	"github.com/nats-io/stan.go"
	"github.com/renegatemaster/wb-l0/cache"
	"github.com/renegatemaster/wb-l0/initializers"
	"github.com/renegatemaster/wb-l0/models"
)

var tpl *template.Template

func init() {
	initializers.LoadEnvVariables()
	initializers.ConnectToDB()
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))
}

var c = cache.NewCache()

func main() {

	// Config
	var (
		clusterID string = os.Getenv("CLUSTER_ID")
		clientID  string = os.Getenv("CLIENT_ID")
		channel   string = os.Getenv("CHANNEL")
		URL       string = stan.DefaultNatsURL
		qgroup    string = os.Getenv("Q_GROUP")
		durable   string = os.Getenv("DURABLE")
	)

	getData()
	defer initializers.DB.Close()

	// Connection to NATS
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL(URL),
		stan.SetConnectionLostHandler(func(_ stan.Conn, reason error) {
			log.Fatalf("Connection lost, reason: %v", reason)
		}))
	if err != nil {
		log.Fatalf("Can't connect: %v.\nMake sure a NATS Streaming Server is running at: %s", err.Error(), URL)
	}
	log.Printf("Connected to %s clusterID: [%s] clientID: [%s]\n", URL, clusterID, clientID)

	// Subscriber
	_, err = sc.QueueSubscribe(channel, qgroup, messageHandler(), stan.DurableName(durable))
	if err != nil {
		sc.Close()
		log.Fatal(err.Error())
	}
	log.Printf("Listening on [%s], clientID=[%s], qgroup=[%s] durable=[%s]\n", channel, clientID, qgroup, durable)

	// http-server
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/orders/:uid", checkCache(getOrder))
	router.POST("/process", processor)

	err = http.ListenAndServe(":8080", router)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Server running on :8080")

	w := sync.WaitGroup{}
	w.Add(1)
	w.Wait()
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tpl.ExecuteTemplate(w, "index.gohtml", nil)
}

// Adds data to cache from DB
func getData() {

	// Requesting orders from DB
	query := "SELECT * FROM orders;"
	rows, err := initializers.DB.Query(query)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	defer rows.Close()
	orders := []models.SimpleOrder{}

	// Adding data to cache
	for rows.Next() {
		order := models.SimpleOrder{}
		err = rows.Scan(&order.Uid, &order.Data)
		if err != nil {
			panic(err)
		}
		orders = append(orders, order)
	}

	for _, order := range orders {
		c.Update(order.Uid, order)
	}
	log.Println("Data added to cache")
}

func processor(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	uid := r.FormValue("order_uid")
	res, ok := c.Read(uid)
	if ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
		return
	}
	log.Printf("From Controller, uid=[%s]", uid)

	// Get entity by pk
	query := "SELECT * FROM orders WHERE uid=$1"
	row := initializers.DB.QueryRow(query, uid)
	order := models.SimpleOrder{}
	err := row.Scan(&order.Uid, &order.Data)
	if err != nil {
		panic(err)
	}

	c.Update(order.Uid, order)
	w.Header().Set("Content-Type", "application/json")
	w.Write(order.Data)
}

// Gets order from DB
func getOrder(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	// Get pk from req
	uid := p.ByName("uid")

	// Get entity by pk
	query := "SELECT * FROM orders WHERE uid=$1"
	row := initializers.DB.QueryRow(query, uid)
	order := models.SimpleOrder{}
	err := row.Scan(&order.Uid, &order.Data)
	if err != nil {
		panic(err)
	}

	c.Update(order.Uid, order)
	w.Header().Set("Content-Type", "application/json")
	w.Write(order.Data)
}

// Wrapper for getOrder() to get order from cache
func checkCache(f httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		uid := p.ByName("uid")
		res, ok := c.Read(uid)
		if ok {
			w.Header().Set("Content-Type", "application/json")
			w.Write(res)
			return
		}
		log.Printf("From Controller, uid=[%s]", uid)
		f(w, r, p)
	}
}

func validateJSON(content []byte) (uid string, data []byte, err error) {

	var order models.Order

	err = json.Unmarshal(content, &order)
	if err != nil {
		return "", nil, err
	}

	uid = order.OrderUid
	if len(uid) == 0 {
		err := errors.New("recived object has no uid")
		return "", nil, err
	}

	data, err = json.Marshal(&order)
	if err != nil {
		return uid, data, err
	}

	return uid, data, nil
}

func addToDB(uid string, data []byte) error {
	query := "INSERT INTO orders (uid, data) VALUES($1, $2);"

	// Adding new entity
	_, err := initializers.DB.Exec(query, uid, data)
	if err != nil {
		return err
	}

	return nil
}

// Handles messages from Nats Streaming
func messageHandler() stan.MsgHandler {
	return func(m *stan.Msg) {
		// Validation
		uid, data, err := validateJSON(m.Data)
		if err != nil {
			log.Println(err.Error())
			return
		}
		log.Printf("Message with uid=[%s] validated", uid)

		// Adding to database
		err = addToDB(uid, data)
		if err != nil {
			log.Println(err.Error())
			return
		}

		// Adding to cache
		order := models.SimpleOrder{Uid: uid, Data: data}
		c.Update(uid, order)
		log.Println("Message handled")
	}
}
