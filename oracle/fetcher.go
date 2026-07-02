package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

type PriceSource struct {
	Name string
	URL  string
}

type Fetcher struct {
	sources []PriceSource
	client  *http.Client
}

func NewFetcher(sources []PriceSource) *Fetcher {
	return &Fetcher{
		sources: sources,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// FetchPrice queries all configured sources in parallel,
// takes the median of received values, and returns an error
// if fewer than 2 sources respond.
func (f *Fetcher) FetchPrice(ctx context.Context) (uint64, error) {
	ch := make(chan uint64, len(f.sources))

	for _, source := range f.sources {
		go func(src PriceSource) {
			price, err := f.querySource(ctx, src.URL)
			if err == nil {
				ch <- price
			}
		}(source)
	}

	var prices []uint64
	timeout := time.After(3 * time.Second)

Loop:
	for i := 0; i < len(f.sources); i++ {
		select {
		case p := <-ch:
			prices = append(prices, p)
		case <-timeout:
			break Loop
		}
	}

	if len(prices) < 2 {
		return 0, fmt.Errorf("insufficient responding sources: got %d, need at least 2", len(prices))
	}

	// Calculate median of retrieved prices
	sort.Slice(prices, func(i, j int) bool { return prices[i] < prices[j] })
	n := len(prices)
	if n%2 == 1 {
		return prices[n/2], nil
	}
	return (prices[n/2-1] + prices[n/2]) / 2, nil
}

type jsonResponse struct {
	Price uint64 `json:"price"`
}

func (f *Fetcher) querySource(ctx context.Context, url string) (uint64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("http error: status %d", resp.StatusCode)
	}

	var r jsonResponse
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return 0, err
	}
	return r.Price, nil
}
