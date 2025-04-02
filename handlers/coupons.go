package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
)

// ErrorResponse represents a standardized error response format
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// SendJSONError sends a standardized error response as JSON
func SendJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResp := ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Status:  status,
	}

	log.Printf("[ERROR] %d %s: %s", status, http.StatusText(status), message)

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ==== Coupon Model ====

type Coupon struct {
	Title      string `json:"title"`
	Link       string `json:"link"`
	CouponCode string `json:"coupon_code"`
	ExpiryDate string `json:"expiry_date"`
	Store      string `json:"store"`
}

// ==== Routing ====

func SetupCouponRoutes() {
	http.HandleFunc("/coupons", GetOnly(GetCouponsHandler))
}

func GetCouponsHandler(w http.ResponseWriter, r *http.Request) {
	storeFilter := r.URL.Query().Get("store")

	var coupons []Coupon
	c := CloneCollectorWithRetry()

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed with response code %d: %v",
			r.Request.URL, r.StatusCode, err)
	})

	c.OnHTML("div.coupons div.coupon", func(e *colly.HTMLElement) {
		coupon := Coupon{
			Title:      strings.TrimSpace(e.ChildText(".coupon__title a")),
			Link:       "https://gun.deals" + strings.TrimSpace(e.ChildAttr(".coupon__title a", "href")),
			CouponCode: strings.TrimSpace(e.ChildText(".coupon__code")),
			Store:      strings.TrimSpace(e.ChildText(".coupon__pills span:nth-child(2) span:nth-child(2)")),
			ExpiryDate: strings.TrimSpace(e.ChildText(".coupon__pills span:first-child span:nth-child(2)")),
		}

		if storeFilter == "" || strings.EqualFold(coupon.Store, storeFilter) {
			coupons = append(coupons, coupon)
		}
	})

	err := c.Visit("https://gun.deals/coupons/latest")
	if err != nil {
		log.Printf("Failed to fetch coupons: %v", err)
		SendJSONError(w, "Failed to fetch coupons. The site may be blocking scraping.", http.StatusInternalServerError)
		return
	}

	c.Wait()

	if len(coupons) == 0 {
		message := "No coupons found"
		if storeFilter != "" {
			message += " for store: " + storeFilter
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  http.StatusNotFound,
			"message": message,
			"store":   storeFilter,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(coupons); err != nil {
		SendJSONError(w, "Failed to encode coupon response", http.StatusInternalServerError)
	}
}
