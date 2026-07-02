package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetcherFetchPrice(t *testing.T) {
	// Setup mock HTTP price servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"price": 60000}`)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"price": 62000}`)
	}))
	defer server2.Close()

	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"price": 61000}`)
	}))
	defer server3.Close()

	sources := []PriceSource{
		{Name: "src1", URL: server1.URL},
		{Name: "src2", URL: server2.URL},
		{Name: "src3", URL: server3.URL},
	}

	fetcher := NewFetcher(sources)
	price, err := fetcher.FetchPrice(context.Background())
	if err != nil {
		t.Fatalf("Expected successful price fetch, got error: %v", err)
	}

	// Median of {60000, 61000, 62000} should be 61000
	if price != 61000 {
		t.Errorf("Expected median price 61000, got %d", price)
	}
}

func TestFetcherInsufficientSources(t *testing.T) {
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"price": 60000}`)
	}))
	defer server1.Close()

	sources := []PriceSource{
		{Name: "src1", URL: server1.URL},
		{Name: "src2", URL: "http://localhost:1111/bad_url_that_fails"},
	}

	fetcher := NewFetcher(sources)
	_, err := fetcher.FetchPrice(context.Background())
	if err == nil {
		t.Error("Expected error due to insufficient responding sources")
	}
}
