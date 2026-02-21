package infrastructure

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/brkss/dextrace/internal/domain"
)

type NightscoutRepository struct {
	nightscoutURL string
	token         string
}

func NewNightscoutRepository(URL string, token string) *NightscoutRepository {
	return &NightscoutRepository{
		nightscoutURL: URL,
		token:         token,
	}
}

func (r *NightscoutRepository) PushData(data []domain.GetDataResponse) error {
	existingEntries, err := r.fetchLatestEntries()
	if err != nil {
		return err
	}

	var latestTimestamp time.Time
	if len(existingEntries) > 0 {
		latestTimestamp, _ = time.Parse(time.RFC3339, existingEntries[0].DateString)
	}

	var newEntries []domain.NightscoutEntry
	for _, d := range data {
		t, err := time.Parse(time.RFC3339, d.Timestamp)
		if err != nil {
			timestampInt := int64(0)
			_, err = fmt.Sscanf(d.Timestamp, "%d", &timestampInt)
			if err != nil {
				continue
			}
			t = time.Unix(0, timestampInt*int64(time.Millisecond))
		}

		if t.After(latestTimestamp) {
			newEntries = append(newEntries, domain.NightscoutEntry{
				Type:       "sgv",
				SGV:        d.Value,
				Date:       t.UnixNano() / int64(time.Millisecond),
				DateString: d.Timestamp,
			})
		}
	}

	if len(newEntries) == 0 {
		return nil
	}

	payload, err := json.Marshal(newEntries)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", r.nightscoutURL+"/api/v1/entries.json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	r.addSecretHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to push data, status: %s, body: %s", resp.Status, string(body))
}

func (r *NightscoutRepository) GetLastRecord() (*domain.NightscoutEntry, error) {
	existingEntries, err := r.fetchLatestEntries()
	if err != nil {
		return nil, err
	}

	if len(existingEntries) > 0 {
		return &existingEntries[0], nil
	}

	return nil, nil
}

func (r *NightscoutRepository) fetchLatestEntries() ([]domain.NightscoutEntry, error) {
	req, err := http.NewRequest("GET", r.nightscoutURL+"/api/v1/entries.json?count=1", nil)
	if err != nil {
		return nil, err
	}
	r.addSecretHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch latest entries, status: %s, body: %s", resp.Status, string(body))
	}

	var existingEntries []domain.NightscoutEntry
	if err := json.Unmarshal(body, &existingEntries); err != nil {
		return nil, fmt.Errorf("failed to decode latest entries: %w; body: %s", err, string(body))
	}

	return existingEntries, nil
}

func (r *NightscoutRepository) addSecretHeader(req *http.Request) {
	if r.token == "" {
		return
	}

	h := sha1.New()
	h.Write([]byte(r.token))
	hashed := hex.EncodeToString(h.Sum(nil))
	req.Header.Set("api-secret", hashed)
}
