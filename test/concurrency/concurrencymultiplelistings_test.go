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

func TestConcurrentBookingsMultipleListings(t *testing.T) {
	type payload struct {
		ListingID       string `json:"listing_id"`
		UserID          string `json:"user_id"`
		CheckIn         string `json:"check_in"`
		CheckOut        string `json:"check_out"`
		PaymentMethodID string `json:"payment_method_id"`
	}

	baseURL := os.Getenv("BOOKING_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/api/v1/bookings"
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 2 Valid Seeded Properties from your SQL schema
	listings := []string{
		"11111111-1111-4111-8111-111111111111", // seed-listing-1
		"22222222-2222-4222-8222-222222222222", // seed-listing-2
	}

	const usersPerListing = 5
	totalRequests := len(listings) * usersPerListing

	var wg sync.WaitGroup
	errs := make(chan error, totalRequests)

	var successCount int32
	var conflictCount int32

	for lIdx, listingID := range listings {
		for uIdx := 1; uIdx <= usersPerListing; uIdx++ {
			wg.Add(1)

			go func(lIdx int, uIdx int, lID string) {
				defer wg.Done()

				// EXACT SAME payload values as your working single-listing test
				body := payload{
					ListingID:       lID,
					UserID:          "22222222-2222-4222-8222-222222222222",
					CheckIn:         "2026-08-10",
					CheckOut:        "2026-08-12",
					PaymentMethodID: "pm_test",
				}

				data, err := json.Marshal(body)
				if err != nil {
					errs <- fmt.Errorf("marshal payload: %w", err)
					return
				}

				req, err := http.NewRequest(http.MethodPost, baseURL, bytes.NewReader(data))
				if err != nil {
					errs <- fmt.Errorf("build request: %w", err)
					return
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Idempotency-Key", fmt.Sprintf("multi-test-prop%d-user%d", lIdx+1, uIdx))

				resp, err := client.Do(req)
				if err != nil {
					errs <- fmt.Errorf("request prop %d user %d: %w", lIdx+1, uIdx, err)
					return
				}
				defer resp.Body.Close()

				t.Logf("[Property %d | User %d] Status: %d", lIdx+1, uIdx, resp.StatusCode)

				switch resp.StatusCode {
				case http.StatusOK, http.StatusCreated:
					atomic.AddInt32(&successCount, 1)
				case http.StatusConflict, http.StatusUnprocessableEntity:
					atomic.AddInt32(&conflictCount, 1)
				default:
					if resp.StatusCode >= http.StatusInternalServerError {
						errs <- fmt.Errorf("request prop %d user %d failed with server error: %s", lIdx+1, uIdx, resp.Status)
					}
				}
			}(lIdx, uIdx, listingID)
		}
	}

	wg.Wait()
	close(errs)

	var details []string
	for err := range errs {
		if err != nil {
			details = append(details, err.Error())
		}
	}

	if len(details) == totalRequests {
		t.Skipf("booking service unavailable at %s", baseURL)
	}

	if len(details) > 0 {
		t.Fatalf("unexpected concurrent booking failures: %v", details)
	}

	// ASSERTION: Exactly 1 booking per property must succeed (2 total successes out of 10 requests)
	expectedSuccesses := int32(len(listings))
	if successCount != expectedSuccesses {
		t.Fatalf("Multi-property lock check failed! Expected %d total successes (1 per listing), but got %d (conflicts: %d)",
			expectedSuccesses, successCount, conflictCount)
	}

	t.Logf("Success! %d properties booked independently (%d successes, %d conflicts). Lock granularity is correct!",
		len(listings), successCount, conflictCount)
}