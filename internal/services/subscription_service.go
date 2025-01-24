package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/umit144/subscription-worker/internal/models"
	"github.com/umit144/subscription-worker/internal/repositories"
)

type SubscriptionService interface {
	ProcessExpiredSubscriptions() error
}

type subscriptionService struct {
	appCredRepo      *repositories.ApplicationCredentialsRepository
	deviceRepo       *repositories.DeviceRepository
	subscriptionRepo *repositories.SubscriptionRepository
	redis            *redis.Client
	batchSize        int
}

func NewSubscriptionService(
	appCredRepo *repositories.ApplicationCredentialsRepository,
	deviceRepo *repositories.DeviceRepository,
	subscriptionRepo *repositories.SubscriptionRepository,
	redis *redis.Client,
	batchSize int,
) SubscriptionService {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &subscriptionService{
		appCredRepo:      appCredRepo,
		deviceRepo:       deviceRepo,
		subscriptionRepo: subscriptionRepo,
		redis:            redis,
		batchSize:        batchSize,
	}
}

type ReceiptResponse struct {
	Status     bool   `json:"status"`
	ExpireDate string `json:"expire-date"`
}

func (s *subscriptionService) ProcessExpiredSubscriptions() error {
	subs, err := s.subscriptionRepo.GetExpiredSubscriptions()
	if err != nil {
		return fmt.Errorf("fetching expired subscriptions: %w", err)
	}

	if len(subs) == 0 {
		return nil
	}

	appIDs, deviceIDs := extractUniqueIDs(subs)
	creds, devices, err := s.fetchRelatedData(appIDs, deviceIDs)
	if err != nil {
		return fmt.Errorf("fetching related data: %w", err)
	}

	return s.processSubscriptions(subs, creds, devices)
}

func (s *subscriptionService) fetchRelatedData(
	appIDs, deviceIDs []uint64,
) (map[uint64]models.ApplicationCredentials, map[uint64]models.Device, error) {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		errors []error
	)

	credMap := make(map[uint64]models.ApplicationCredentials)
	deviceMap := make(map[uint64]models.Device)

	wg.Add(2)

	go func() {
		defer wg.Done()
		creds, err := s.appCredRepo.FetchByIds(appIDs)
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("fetching credentials: %w", err))
			mu.Unlock()
			return
		}

		for _, cred := range creds {
			credMap[cred.ID] = cred
		}
	}()

	go func() {
		defer wg.Done()
		devices, err := s.deviceRepo.FetchByIds(deviceIDs)
		if err != nil {
			mu.Lock()
			errors = append(errors, fmt.Errorf("fetching devices: %w", err))
			mu.Unlock()
			return
		}

		for _, dev := range devices {
			deviceMap[dev.ID] = dev
		}
	}()

	wg.Wait()

	if len(errors) > 0 {
		return nil, nil, errors[0]
	}

	return credMap, deviceMap, nil
}

func (s *subscriptionService) processSubscriptions(
	subs []models.Subscription,
	creds map[uint64]models.ApplicationCredentials,
	devices map[uint64]models.Device,
) error {
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	for _, sub := range subs {
		cred, ok := creds[sub.ApplicationID]
		if !ok {
			continue
		}

		dev, ok := devices[sub.DeviceID]
		if !ok {
			continue
		}

		log.Printf("[INFO] Processing subscription - ID: %d | App: %s | Device: %d", sub.ID, cred.Username, dev.ID)

		resp, err := s.validateReceipt(client, sub.Receipt, cred)
		if err != nil {
			log.Fatalf("Error validating receipt for subscription %d: %v\n", sub.ID, err)
			continue
		}

		event, err := s.updateSubscription(sub.ID, resp)
		if err != nil {
			log.Fatalf("Error updating subscription %d: %v\n", sub.ID, err)
			continue
		}

		err = s.publishSubscriptionEvent(sub, event)
		if err != nil {
			log.Fatalf("Error publishing event : %v\n", err)
			continue
		}
	}

	return nil
}

func (s *subscriptionService) validateReceiptWithRetry(client *http.Client, receipt string, cred models.ApplicationCredentials) (*ReceiptResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err := s.validateReceipt(client, receipt, cred)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		time.Sleep(time.Second * time.Duration(attempt*5))
	}
	return nil, fmt.Errorf("max retries reached: %w", lastErr)
}

func (s *subscriptionService) validateReceipt(client *http.Client, receipt string, cred models.ApplicationCredentials) (*ReceiptResponse, error) {
	if len(receipt) >= 2 {
		if num, err := strconv.Atoi(receipt[len(receipt)-2:]); err == nil && num%6 == 0 {
			return s.validateReceiptWithRetry(client, receipt, cred)
		}
	}

	body := struct {
		Receipt string `json:"receipt"`
	}{
		Receipt: receipt,
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	var baseUrl string
	switch cred.Platform {
	case "ios":
		baseUrl = os.Getenv("APP_STORE_API")
	case "android":
		baseUrl = os.Getenv("GOOGLE_PLAY_API")
	}

	url := fmt.Sprintf("%s/receipt/validate", baseUrl)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(cred.Username, cred.Password)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var receiptResp ReceiptResponse
	if err := json.NewDecoder(resp.Body).Decode(&receiptResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &receiptResp, nil
}

func (s *subscriptionService) updateSubscription(id uint64, resp *ReceiptResponse) (string, error) {
	if resp.Status {
		expireDate, err := time.Parse("2006-01-02 15:04:05", resp.ExpireDate)
		if err != nil {
			return "", fmt.Errorf("parsing expire date: %w", err)
		}
		err = s.subscriptionRepo.UpdateExpireDate(id, expireDate)
		if err != nil {
			return "", fmt.Errorf("parsing expire date: %w", err)
		}
		return "renewed", nil
	}

	err := s.subscriptionRepo.UpdateStatus(id, false)
	if err != nil {
		return "", fmt.Errorf("parsing expire date: %w", err)
	}
	return "canceled", nil
}

func extractUniqueIDs(subs []models.Subscription) ([]uint64, []uint64) {
	appSet := make(map[uint64]struct{})
	deviceSet := make(map[uint64]struct{})

	for _, sub := range subs {
		appSet[sub.ApplicationID] = struct{}{}
		deviceSet[sub.DeviceID] = struct{}{}
	}

	return mapToSlice(appSet), mapToSlice(deviceSet)
}

func mapToSlice(m map[uint64]struct{}) []uint64 {
	result := make([]uint64, 0, len(m))
	for id := range m {
		result = append(result, id)
	}
	return result
}

func (s *subscriptionService) publishSubscriptionEvent(sub models.Subscription, event string) error {
	log.Printf("publishing event")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data := map[string]interface{}{
		"appId":    sub.ApplicationID,
		"deviceId": sub.DeviceID,
		"event":    event,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling event data: %w", err)
	}

	err = s.redis.Publish(ctx, "notifications.subscription.updated", string(jsonData)).Err()
	if err != nil {
		return fmt.Errorf("publishing event: %w", err)
	}

	log.Printf("event published successfully: %s", event)
	return nil
}
