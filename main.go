package main

import (
	"webscraper/bot"
	"webscraper/scraper"
)

func main() {
	scraper.ScrapeData()
	bot.StartBot()
}
