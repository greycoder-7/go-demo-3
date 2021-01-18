package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"gopkg.in/gookit/color.v1"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jcostabe/go-demo-3/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var cfg *model.Config
var collection *mongo.Collection
var ctx = context.TODO()

var (
	clientConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "active_connections",
		Help: "Number of active client connections",
	}, []string{"service"})

	transactionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_req_processed_total",
		Help: "Total number of HTTP requests processed",
	}, []string{"code", "method"})

	responseTimeHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_req_duration_seconds",
		Help:    "Duration of all HTTP requests",
		Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	}, []string{
		"service",
		"code",
		"method",
		"path",
	})
)

type Book struct {
	ID     string  `bson: "id"`
	Isbn   string  `bson: "isbn"`
	Title  string  `bson: "title"`
	Price  string  `bson: "price"`
	Author *Author `bson: "author"`
}

type Author struct {
	FirstName string `bson:"firstname"`
	LastName  string `bson:"lastname"`
}

func getBooks(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logrus.Infof("Request /api/books")
	w.Header().Set("Content-Type", "application/json")

	findOptions := options.Find()
	findOptions.SetLimit(5)

	var results []*Book

	cur, err := collection.Find(ctx, bson.D{{}}, findOptions)
	if err != nil {
		logrus.Fatal(err)
	}

	for cur.Next(ctx) {
		var elem Book
		err := cur.Decode(&elem)
		if err != nil {
			logrus.Fatal(err)
		}

		results = append(results, &elem)
	}

	if err := cur.Err(); err != nil {
		logrus.Fatal(err)
	}

	cur.Close(ctx)

	logrus.Infof("Found multiple documents (array of pointers): %v\n", results)
	json.NewEncoder(w).Encode(results)
	logrus.Infof("Elapsed time of /api/book response: %v", time.Since(start))

}

func getBook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logrus.Infof("Request /api/book")
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	var result Book

	filter := bson.D{{"isbn", params["isbn"]}}

	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		logrus.Infof("Book with ID %v not found", params["isbn"])
	}

	if result.Isbn != "" {
		logrus.Infof("Found a single document: %+v\n", result)
		json.NewEncoder(w).Encode(result)

	} else {
		io.WriteString(w, "Book not found")

	}
	logrus.Infof("Elapsed time of /api/book response: %v", time.Since(start))

}

func createBook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logrus.Infof("Request: Creating a new entry in book store")

	w.Header().Set("Content-Type", "application/json")
	vars := r.URL.Query()

	idParam := vars.Get("id")
	isbnParam := vars.Get("isbn")
	titleParam := vars.Get("title")
	priceParam := vars.Get("price")
	authorNameParam := vars.Get("author_name")
	authorLastameParam := vars.Get("author_lastname")

	book := Book{
		ID:    idParam,
		Isbn:  isbnParam,
		Title: titleParam,
		Price: priceParam,
		Author: &Author{
			FirstName: authorNameParam,
			LastName:  authorLastameParam},
	}

	insertResult, err := collection.InsertOne(ctx, book)
	if err != nil {
		logrus.Fatal(err)
	}

	json.NewEncoder(w).Encode("Book entry created successfully")

	logrus.Infof("Book had been inserted: ", insertResult.InsertedID)
	logrus.Infof("Elapsed time of /api/books response: %v", time.Since(start))

}

func updateBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	logrus.Infof("Request: Update book")

	w.Header().Set("Content-Type", "application/json")
	vars := r.URL.Query()

	idParam := vars.Get("id")
	priceParam := vars.Get("price")

	filter := bson.D{{"id", idParam}}

	updateResult, err := collection.UpdateOne(ctx, filter, bson.D{
		{"$set", bson.D{
			{"price", priceParam},
		}},
	})
	if err != nil {
		logrus.Fatal(err)
	}

	json.NewEncoder(w).Encode("Price updated")

	logrus.Infof("Matched %v documents and updated %v documents.\n", updateResult.MatchedCount, updateResult.ModifiedCount)
}

func deleteBook(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Content-Type", "application/json")
	logrus.Infof("Request: Delete book in book store")

	vars := r.URL.Query()

	isbnParam := vars.Get("isbn")

	deleteResult, err := collection.DeleteMany(ctx, bson.D{{"isbn", isbnParam}})
	if err != nil {
		logrus.Fatal(err)
	}

	json.NewEncoder(w).Encode("Book entry deleted successfully")

	logrus.Infof("Deleted %v documents in the books collection\n", deleteResult.DeletedCount)
	logrus.Infof("Elapsed time of /api/deletion response: %v", time.Since(start))

}

func deleteBooks(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Content-Type", "application/json")
	logrus.Infof("Request: Delete all books")

	deleteResult, err := collection.DeleteMany(ctx, bson.D{{}})
	if err != nil {
		logrus.Fatal(err)
	}

	json.NewEncoder(w).Encode("Book entries deleted successfully")

	logrus.Infof("Deleted %v documents in the books collection\n", deleteResult.DeletedCount)
	logrus.Infof("Elapsed time of /api/deletion response: %v", time.Since(start))

}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	logrus.Infof("Alive method")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	io.WriteString(w, `{"alive": true}`)
	logrus.Infof("Elapsed time of /isAlive: %v", time.Since(start))

}

func getHostInfo(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logrus.Infof("Host info")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	host, _ := os.Hostname()
	io.WriteString(w, host)
	logrus.Infof("Elapsed time of /info: %v", time.Since(start))

}

func getVersion(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	cfg = model.DefaultConfiguration()

	logrus.Infof("Version info")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	io.WriteString(w, cfg.ServiceConfig.Version)
	logrus.Infof("Elapsed time of /version: %v", time.Since(start))

}

func init() {

	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000",
		FullTimestamp:   true,
	})
}

// debug mode for production environment
/*func init() {
    profile := flag.String("profile", "test", "Environment profile")
	if *profile == "dev" {
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000",
			FullTimestamp: true,
		})
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}*/

func initDb(cfg *model.Config) {

	clientOps := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%d", cfg.DatabaseConfig.Host, cfg.DatabaseConfig.Port))
	client, err := mongo.Connect(ctx, clientOps)
	if err != nil {
		logrus.Fatal(err)
	}

	// Check the connections
	err = client.Ping(ctx, nil)
	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Infof("Application connected to database %s at port %d", cfg.DatabaseConfig.Host, cfg.DatabaseConfig.Port)

	collection = client.Database("bookstore").Collection("books")
	insertInitialData()

}

func insertInitialData() {
	book1 := Book{
		ID:     "1",
		Isbn:   "9812005321",
		Title:  "Kubernetes Training",
		Price:  "32$",
		Author: &Author{FirstName: "John", LastName: "Doe"},
	}

	book2 := Book{
		ID:     "2",
		Isbn:   "47192038471",
		Title:  "Kubernetes Training - Part 2",
		Price:  "20$",
		Author: &Author{FirstName: "John", LastName: "Doe"},
	}

	book3 := Book{
		ID:     "3",
		Isbn:   "360123401",
		Title:  "Kubernetes Training - Part 3",
		Price:  "25$",
		Author: &Author{FirstName: "John", LastName: "Doe"},
	}

	books := []interface{}{book1, book2, book3}

	insertManyResult, _ := collection.InsertMany(ctx, books)

	logrus.Infof("Inserted multiple documents: ", insertManyResult.InsertedIDs)

}

func delayedResponse(w http.ResponseWriter, r *http.Request) {

	start := time.Now()
	code := http.StatusOK
	clientConnections.WithLabelValues(cfg.ServiceConfig.Name).Inc()
	defer func() { recordMetrics(r.URL.Path, r.Method, code, start) }()

	randDuration := rand.Intn(30)
	sleep := time.Duration(randDuration) * time.Second
	logrus.Infof("Delayed response with a duration of %v", sleep)
	time.Sleep(sleep)

	params := r.URL.Query()
	message := params.Get("message")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	io.WriteString(w, message)
	logrus.Infof("Elapsed time of %s: %v", r.URL.Path, time.Since(start))

}

func randomError(w http.ResponseWriter, r *http.Request) {

	start := time.Now()
	code := http.StatusOK
	clientConnections.WithLabelValues(cfg.ServiceConfig.Name).Inc()
	defer func() { recordMetrics(r.URL.Path, r.Method, code, start) }()

	switch randErr := rand.Intn(3); randErr {
	case 1:
		code = http.StatusNotFound
		w.WriteHeader(code)
		w.Write([]byte("404 - Not found"))
	case 2:
		code = http.StatusInternalServerError
		w.WriteHeader(code)
		w.Write([]byte("500 - Internal server error"))
	default:
		w.WriteHeader(code)
		w.Write([]byte("200 - It works fine!"))

	}

	logrus.Infof("Elapsed time of %s: %v", r.URL.Path, time.Since(start))
}

func recordMetrics(requestPath string, requestMethod string, statusCode int, start time.Time) {

	code := strconv.Itoa(statusCode)
	transactionsTotal.WithLabelValues(code, requestPath).Inc()

	clientConnections.WithLabelValues(cfg.ServiceConfig.Name).Dec()

	duration := time.Since(start)
	responseTimeHistogram.WithLabelValues(cfg.ServiceConfig.Name, code, requestMethod, requestPath).Observe(duration.Seconds())

}

func initRestService() {

	r := mux.NewRouter()
	logrus.Infof("Application running...")

	// Route handlers / Endpoints
	r.HandleFunc("/isAlive", healthCheckHandler).Methods("GET")
	r.HandleFunc("/info", getHostInfo).Methods("GET")
	r.HandleFunc("/version", getVersion).Methods("GET")

	r.HandleFunc("/echoWithDelay", delayedResponse).Methods("GET")
	r.HandleFunc("/randomError", randomError).Methods("GET")

	r.Path("/metrics").Handler(promhttp.Handler())

	r.HandleFunc("/api/books", getBooks).Methods("GET")
	r.HandleFunc("/api/books/{isbn}", getBook).Methods("GET")
	r.HandleFunc("/api/books", createBook).Methods("POST")
	r.HandleFunc("/api/books/{isbn}", updateBook).Methods("PUT")
	r.HandleFunc("/api/deletebook", deleteBook).Methods("DELETE")
	r.HandleFunc("/api/deletebooks", deleteBooks).Methods("DELETE")

	logrus.Fatal(http.ListenAndServe(":8080", r))

}

func main() {

	cfg = model.DefaultConfiguration()

	color.Blue.Println("Service: " + cfg.ServiceConfig.Name)
	color.Green.Println("Version: " + cfg.ServiceConfig.Version)
	color.Yellow.Println("Config profile: " + cfg.Environment)

	initDb(cfg)
	initRestService()

}
