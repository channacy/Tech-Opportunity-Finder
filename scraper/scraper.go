package scraper

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"webscraper/shared"

	"strings"

	"github.com/gocolly/colly"
	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func ScrapeData() {
	var allOpportunities []shared.Opportunity
	envFile, _ := godotenv.Read(".env")

	envGoogleApplicationCred := envFile["GOOGLE_APPLICATION_CREDENTIALS"]
	docId := envFile["GSHEET_ID"]
	mongoURI := envFile["MONGODB_URI"]

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

	if link, ok := links["Underclassmen Internships"]; ok {
		underClassmenInternships, err := getUnderclassmenInternships(link)
		if err != nil {
			fmt.Println("Issue getting underclassmen internships.")
		} else {
			allOpportunities = append(allOpportunities, underClassmenInternships...)
		}
	}

	if link, ok := links["DEV"]; ok {
		articles, err := getArticles(link)
		if err != nil {
			fmt.Println("Issue getting underclassmen internships.")
		} else {
			allOpportunities = append(allOpportunities, articles...)
		}
	}

	setupMongoDB(mongoURI, allOpportunities)

}

func getUnderclassmenInternships(url string) ([]shared.Opportunity, error) {
	collector := colly.NewCollector()
	var internships []shared.Opportunity

	collector.OnRequest(func(r *colly.Request) {
		// print the url of that request
		fmt.Println("Visiting", r.URL)
	})

	collector.OnHTML("table tr", func(e *colly.HTMLElement) {
		opportunity := shared.Opportunity{}
		if !strings.Contains(e.Text, "⛔") && !strings.Contains(e.Text, "Name") && strings.Contains(e.Text, "✅") {
			opportunity.Name = e.Text
			opportunity.Url = e.ChildAttr("a", "href")
			opportunity.Category = "Undergraduate Underclassmen Internships"
			opportunity.ExpireAfterSeconds = 60
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

func getArticles(url string) ([]shared.Opportunity, error) {
	var articles []shared.Opportunity
	collector := colly.NewCollector()

	collector.OnRequest(func(r *colly.Request) {
		// print the url of that request
		fmt.Println("Visiting", r.URL)
	})

	collector.OnHTML(".crayons-story", func(e *colly.HTMLElement) {
		article := shared.Opportunity{}
		article.Url = "https://dev.to/" + e.ChildAttr("a", "href")
		article.Name = e.ChildText(".crayons-story__hidden-navigation-link")
		article.Category = "Articles"
		article.ExpireAfterSeconds = 60
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

func setupMongoDB(mongoURI string, opportunities []shared.Opportunity) {

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}

	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	if err := client.Database("techOptFinderDB").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	coll := client.Database("techOptFinderDB").Collection("opportunities")

	fmt.Println(opportunities)

	var documents []interface{}
	for _, opp := range opportunities {
		documents = append(documents, opp)
	}
	fmt.Printf("Documents to be inserted: %+v\n", documents)

	result, err := coll.InsertMany(context.TODO(), documents)
	if err != nil {
		panic(err)
	}
	fmt.Println(result)

}
