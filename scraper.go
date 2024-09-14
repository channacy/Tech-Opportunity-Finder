package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"strings"

	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Opportunity struct {
	name     string
	url      string
	category string
}

func main() {
	envFile, _ := godotenv.Read(".env")

	envGoogleApplicationCred := envFile["GOOGLE_APPLICATION_CREDENTIALS"]
	docId := envFile["GSHEET_ID"]

	links := make(map[string]string)
	keyFilePath := flag.String("keyfile", envGoogleApplicationCred, "path to the credentials file")
	flag.Parse()

	ctx := context.Background()
	credentials, err := os.ReadFile(*keyFilePath)
	if err != nil {
		log.Fatal("unable to read key file:", err)
	}

	scopes := []string{
		"https://www.googleapis.com/auth/spreadsheets.readonly",
	}

	config, err := google.JWTConfigFromJSON(credentials, scopes...)
	if err != nil {
		log.Fatal("unable to create JWT configuration:", err)
	}

	srv, err := sheets.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		log.Fatalf("unable to retrieve sheets service: %v", err)
	}

	doc, err := srv.Spreadsheets.Get(docId).Do()
	if err != nil {
		log.Fatalf("unable to retrieve data from document: %v", err)
	}
	fmt.Printf("The title of the doc is: %s\n", doc.Properties.Title)

	val, err := srv.Spreadsheets.Values.Get(docId, "Sheet1!A2:B3").Do()
	if err != nil {
		log.Fatalf("unable to retrieve range from document: %v", err)
	}

	fmt.Printf("Selected major dimension=%v, range=%v\n", val.MajorDimension, val.Range)
	for _, row := range val.Values {
		//The fmt.Sprint() function in Go language formats using the
		//default formats for its operands and returns the resulting string
		links[fmt.Sprint(row[0])] = fmt.Sprint(row[1])
	}

	fmt.Println(links)

	if link, ok := links["Underclassmen Internships"]; ok {
		underClassmenInternships, err := getUnderclassmenInternships(link)
		if err != nil {
			fmt.Println("Issue getting underclassmen internships.")
		} else {
			setupMongoDB(underClassmenInternships)
		}
	}
	// if link, ok := links["DEV"]; ok {
	// 	getArticles(link)
	// }

	// var opportunities []Opportunity

	// underclassmenInternships, err := getUnderclassmenInternships(url)
	// if err != nil {
	// 	fmt.Println("Issue getting underclassmen internships.")
	// }

	// articles, err := getArticles(url)
	// if err != nil {
	// 	fmt.Println("Issue getting underclassmen internships.")
	// }

	// opportunities = append(opportunities, underclassmenInternships...)
	// opportunities = append(opportunities, articles...)

	// hackathons, err := getHackathons(url)
	// if err != nil {
	// 	fmt.Println("Issue getting hackathons.")
	// }
	// opportunities = append(opportunities, hackathons...)

	//store in database
	//fmt.Println(underclassmenInternships)
	// opportunities = append(opportunities, articles...)
	// fmt.Println(opportunities)
}

func getUnderclassmenInternships(url string) ([]Opportunity, error) {
	collector := colly.NewCollector()
	var internships []Opportunity

	collector.OnRequest(func(r *colly.Request) {
		// print the url of that request
		fmt.Println("Visiting", r.URL)
	})

	collector.OnHTML("table tr", func(e *colly.HTMLElement) {
		opportunity := Opportunity{}
		if !strings.Contains(e.Text, "⛔") && !strings.Contains(e.Text, "Name") {
			// fmt.Println(e.Text)
			// fmt.Println(e.ChildAttr("a", "href"))
			// fmt.Println("--")
			opportunity.name = e.Text
			opportunity.url = e.ChildAttr("a", "href")
			opportunity.category = "Undergraduate Underclassmen Internships"
		}
		internships = append(internships, opportunity)
	})

	collector.OnResponse(func(r *colly.Response) {
		fmt.Println("Got a response from", r.Request.URL)
	})
	collector.OnError(func(r *colly.Response, e error) {
		fmt.Println("Error occurred!:", e)
	})

	collector.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished scraping", r.Request.URL)
	})

	collector.Visit(url)

	return internships, nil
}

func getArticles(url string) ([]Opportunity, error) {
	var articles []Opportunity
	collector := colly.NewCollector()

	collector.OnRequest(func(r *colly.Request) {
		// print the url of that request
		fmt.Println("Visiting", r.URL)
	})

	collector.OnHTML(".crayons-story", func(e *colly.HTMLElement) {
		article := Opportunity{}
		article.url = "https://dev.to/" + e.ChildAttr("a", "href")
		article.name = e.ChildText(".crayons-story__hidden-navigation-link")
		// fmt.Println(article.name)
		// fmt.Println(article.url)
		// fmt.Println("----")
		articles = append(articles, article)
	})

	collector.OnResponse(func(r *colly.Response) {
		fmt.Println("Got a response from", r.Request.URL)
	})
	collector.OnError(func(r *colly.Response, e error) {
		fmt.Println("Error occurred!:", e)
	})

	// triggered once scraping is done (e.g., write the data to a CSV file)
	collector.OnScraped(func(r *colly.Response) {
		fmt.Println("Finished scraping", r.Request.URL)
	})

	collector.Visit(url)
	return articles, nil

}

func setupMongoDB(opportunities []Opportunity) {
	// Set client options
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	// Connect to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Check the connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")
	// Access a MongoDB collection
	collection := client.Database("techOptFinderDB").Collection("opportunities")

	// Define a slice of interface{}
	var docs []interface{}
	// Convert each Opportunity struct to an interface{}
	for _, op := range opportunities {
		docs = append(docs, op)
	}

	// Insert a document
	_, err = collection.InsertMany(context.Background(), docs)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Inserted document into collection!")
	// Find a document
	var result Opportunity
	err = collection.FindOne(context.Background(), nil).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found document: %+v\n", result)

	// Disconnect from MongoDB
	err = client.Disconnect(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connection to MongoDB closed.")

}

// func getHackathons(url string) ([]Opportunity, error) {
// 	var hackathons []Opportunity
// 	collector := colly.NewCollector()

// 	collector.OnRequest(func(r *colly.Request) {
// 		// print the url of that request
// 		fmt.Println("Visiting", r.URL)
// 	})

// 	collector.OnHTML("div.hackathon-tile", func(e *colly.HTMLElement) {
// 		// hackathonTitle := e.ChildText("h3.mb-4")
// 		// fmt.Println("Hackathon Title:", hackathonTitle)
// 		// title := e.ChildText("h2")
// 		fmt.Println(e.ChildText("a"))
// 	})

// 	collector.OnResponse(func(r *colly.Response) {
// 		fmt.Println("Got a response from", r.Request.URL)
// 	})
// 	collector.OnError(func(r *colly.Response, e error) {
// 		fmt.Println("Error occurred!:", e)
// 	})

// 	collector.OnScraped(func(r *colly.Response) {
// 		fmt.Println("Finished", r.Request.URL)
// 	})

// 	collector.Visit("https://devpost.com/hackathons")

// 	return hackathons, nil
// }
