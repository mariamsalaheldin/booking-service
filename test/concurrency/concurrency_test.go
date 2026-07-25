package concurrency

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentBookings(t *testing.T) {
	type payload struct {
		ListingID       string `json:"listing_id"`
		UserID          string `json:"user_id"`
		CheckIn         string `json:"check_in"`
		CheckOut        string `json:"check_out"`
		PaymentMethodID string `json:"payment_method_id"`
	}

	body := payload{
		ListingID:       "88888888-8888-4888-8888-888888888888",
		UserID:          "22222222-2222-4222-8222-222222222222",
		CheckIn:         "2026-08-10",
		CheckOut:        "2026-08-12",
		PaymentMethodID: "pm_test",
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	baseURL := os.Getenv("BOOKING_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/api/v1/bookings"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	const concurrency = 5

	var wg sync.WaitGroup
	errs := make(chan error, concurrency)

	// Counters for tracking lock outcomes
	var successCount int32
	var conflictCount int32

	for i := 1; i <= concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewReader(data))
			if err != nil {
				errs <- fmt.Errorf("build request %d: %w", i, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", fmt.Sprintf("concurrency-test-%d", i))

			resp, err := client.Do(req)
			if err != nil {
				errs <- fmt.Errorf("request %d: %w", i, err)
				return
			}
			defer resp.Body.Close()

			t.Logf("[%d] Response Status: %d", i, resp.StatusCode)

			switch resp.StatusCode {
			case http.StatusOK, http.StatusCreated:
				atomic.AddInt32(&successCount, 1)
			case http.StatusConflict, http.StatusUnprocessableEntity:
				atomic.AddInt32(&conflictCount, 1)
			default:
				if resp.StatusCode >= http.StatusInternalServerError {
					errs <- fmt.Errorf("request %d failed with server error: %s", i, resp.Status)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	var details []string
	for err := range errs {
		if err != nil {
			details = append(details, err.Error())
		}
	}

	// 1. If server was completely unreachable for all requests, skip gracefully
	if len(details) == concurrency {
		t.Skipf("booking service unavailable at %s: %s", baseURL, details[0])
	}

	// 2. Fail if unexpected network/500 errors occurred
	if len(details) > 0 {
		t.Fatalf("unexpected concurrent booking failures: %v", details)
	}

	// 3. ASSERTION: Exactly 1 request must win the reservation, and remaining should be rejected
	if successCount != 1 {
		t.Fatalf("Concurrency check failed! Expected exactly 1 successful booking, but got %d (conflicts: %d)", successCount, conflictCount)
	}

	t.Logf("Success! Exactly 1 booking succeeded and %d were correctly blocked.", conflictCount)
}
