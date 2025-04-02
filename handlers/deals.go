package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// ==== Models ====

type Product struct {
	Image          string                 `json:"image"`
	Title          string                 `json:"title"`
	Details        map[string]interface{} `json:"details,omitempty"`
	Price          string                 `json:"price"`
	Price_Addition string                 `json:"price_addition,omitempty"`
	Merchant       string                 `json:"merchant,omitempty"`
	Link           string                 `json:"link"`
}

type InventoryListing struct {
	StoreName  string `json:"store_name"`
	Link       string `json:"link"`
	Price      string `json:"price"`
	Shipping   string `json:"shipping"`
	OutOfStock bool   `json:"out_of_stock"`
}

type ProductPage struct {
	Image       string                 `json:"image"`
	Title       string                 `json:"title"`
	Details     map[string]interface{} `json:"details,omitempty"`
	AllListings []InventoryListing     `json:"all_listings"`
}

// ==== Helpers ====

func RespondWithJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	http.Error(w, message, code)
}

func GetOnly(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			RespondWithError(w, http.StatusMethodNotAllowed, "Method not allowed. Only GET requests are supported.")
			return
		}
		handler(w, r)
	}
}

func NewGunDealsCollector() *colly.Collector {
	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		colly.AllowURLRevisit(),
		colly.MaxDepth(1),
	)

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 2 * time.Second,
	})

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
		r.Headers.Set("Referer", "https://www.google.com/")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Sec-Ch-Ua", "\"Chromium\";v=\"122\", \"Not(A:Brand\";v=\"24\", \"Google Chrome\";v=\"122\"")
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", "\"Windows\"")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Sec-Fetch-User", "?1")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Cache-Control", "max-age=0")
		r.Headers.Set("Connection", "keep-alive")
	})

	return c
}

// ==== Scraping ====

func ScrapeGunDealsTiles(url string) ([]Product, error) {
	var products []Product
	c := CloneCollectorWithRetry()

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed: %v", r.Request.URL, err)
	})

	c.OnHTML(".tile-container", func(e *colly.HTMLElement) {
		product := Product{
			Image:          e.ChildAttr(".tile__image", "src"),
			Title:          e.ChildText(".tile__title span span"),
			Price:          e.ChildText(".tile__price"),
			Price_Addition: e.ChildText(".tile__price-addition"),
			Merchant:       e.ChildText(".tile__go-to-store a span"),
			Link:           "https://gun.deals" + e.ChildAttr(".tile__go-to-store a", "href"),
		}
		products = append(products, product)
	})

	err := c.Visit(url)
	if err != nil {
		return nil, err
	}

	c.Wait()

	if len(products) == 0 {
		return nil, fmt.Errorf("no products found - site may be blocking scraping")
	}

	return products, nil
}

// ==== Route Handlers ====

func StaticDealsHandler(url string) http.HandlerFunc {
	return GetOnly(func(w http.ResponseWriter, r *http.Request) {
		products, err := ScrapeGunDealsTiles(url)
		if err != nil {
			log.Println("Error in static deal handler:", err)
			RespondWithError(w, http.StatusServiceUnavailable, "Failed to fetch deals: "+err.Error())
			return
		}
		RespondWithJSON(w, products)
	})
}

func SearchDeals(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing search query. Use ?q=your-search-term")
		return
	}

	page := r.URL.Query().Get("page")
	if page == "" {
		page = "0"
	} else if page != "0" {
		if pageNum, err := strconv.Atoi(page); err == nil {
			page = strconv.Itoa(pageNum - 1)
		}
	}

	var products []Product
	c := CloneCollectorWithRetry()

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed: %v", r.Request.URL, err)
	})

	c.OnHTML(".product-details-tile", func(e *colly.HTMLElement) {
		product := Product{
			Image: e.ChildAttr(".product-details-tile__image-container__image", "src"),
			Title: e.ChildAttr(".product-details-tile__title__link", "title"),
			Price: e.ChildText(".price-tag"),
		}

		details := make(map[string]interface{})
		e.ForEach(".product-details-tile__info__specs", func(_ int, el *colly.HTMLElement) {
			name := strings.TrimSuffix(el.ChildText("span:first-child"), ":")
			value := el.ChildText("span:last-child")
			if name != "" && value != "" {
				details[name] = value
			}
		})
		product.Details = details

		if upc, ok := details["UPC"]; ok {
			domain := os.Getenv("DOMAIN")
			product.Link = fmt.Sprintf("http://%s/product/%s", domain, upc)
		}

		products = append(products, product)
	})

	searchURL := fmt.Sprintf("https://gun.deals/search/apachesolr_search/%s?result_type=product&page=%s", query, page)
	err := c.Visit(searchURL)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to search deals: "+err.Error())
		return
	}

	if len(products) == 0 {
		RespondWithJSON(w, map[string]string{
			"message": "No products found for your search query",
			"query":   query,
		})
		return
	}

	RespondWithJSON(w, products)
}

func GetProduct(w http.ResponseWriter, r *http.Request) {
	upc := strings.TrimPrefix(r.URL.Path, "/product/")
	if upc == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing product UPC. Use /product/product-upc")
		return
	}

	productPage := ProductPage{}
	c := CloneCollectorWithRetry()

	c.OnHTML(".product-basic-info-card__image img", func(e *colly.HTMLElement) {
		productPage.Image = e.Attr("src")
	})

	c.OnHTML(".product-basic-info-card__main-content__title h1", func(e *colly.HTMLElement) {
		productPage.Title = strings.TrimSpace(e.Text)
	})

	c.OnHTML(".specifications", func(e *colly.HTMLElement) {
		details := make(map[string]interface{})
		e.ForEach("div", func(_ int, el *colly.HTMLElement) {
			name := strings.TrimSpace(el.ChildText("dd"))
			value := strings.TrimSpace(el.ChildText("dt"))
			if name != "" && value != "" {
				details[name] = value
			}
		})
		productPage.Details = details
	})

	c.OnHTML(".price-compare-table tbody tr", func(e *colly.HTMLElement) {
		if e.Attr("class") == "header" {
			return
		}

		listing := InventoryListing{
			StoreName:  strings.TrimSpace(e.ChildText("td.price-compare-table__store-column div a span")),
			Price:      strings.TrimSpace(e.ChildText("td.price-compare-table__price-column")),
			Shipping:   strings.TrimSpace(e.ChildText("td.price-compare-table__shipping-column")),
			Link:       e.ChildAttr("td.price-compare-table__store-column div a", "href"),
			OutOfStock: strings.Contains(strings.ToLower(e.Attr("class")), "is-oos"),
		}
		productPage.AllListings = append(productPage.AllListings, listing)
	})

	productURL := fmt.Sprintf("https://gun.deals/search/apachesolr_search/%s", upc)
	err := c.Visit(productURL)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to fetch product details: "+err.Error())
		return
	}
	c.Wait()

	if productPage.Title == "" {
		RespondWithError(w, http.StatusNotFound, "Product not found")
		return
	}

	RespondWithJSON(w, productPage)
}

func CategoryDeals(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/category/")
	category := strings.TrimSpace(path)

	if category == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing category. Use /category/<category-slug> (e.g., /category/hand-guns)")
		return
	}

	url := fmt.Sprintf("https://gun.deals/category/%s", category)
	products, err := ScrapeGunDealsTiles(url)
	if err != nil {
		log.Printf("Error fetching category '%s': %v", category, err)
		RespondWithError(w, http.StatusServiceUnavailable, "Failed to fetch category deals: "+err.Error())
		return
	}

	RespondWithJSON(w, products)
}

// ==== Routing ====

func SetupDealsRoutes() {
	staticRoutes := map[string]string{
		"/relevant": "https://gun.deals/", // Get relevant deals from home page
		"/today":    "https://gun.deals/today",
	}

	for route, url := range staticRoutes {
		http.HandleFunc(route, StaticDealsHandler(url))
	}

	http.HandleFunc("/search", GetOnly(SearchDeals))
	http.HandleFunc("/product/", GetOnly(GetProduct))
	http.HandleFunc("/category/", GetOnly(CategoryDeals))
}
