package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestQueryPrometheus(t *testing.T) {
	mockResponse := `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [
				{
					"metric": {"__name__": "spanner_cpu_utilization"},
					"value": [1710000000, "78.45"]
				}
			]
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient(server.URL, 2*time.Second)
	val, err := client.Query(context.Background(), "spanner_cpu_utilization")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if val != 78.45 {
		t.Fatalf("expected 78.45, got %f", val)
	}
}
