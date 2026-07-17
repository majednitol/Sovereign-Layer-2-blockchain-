//go:build dev

package relayer

import "fmt"

func (s *HorcruxSignerClient) Sign(hash []byte) ([]byte, error) {
	fmt.Printf("[HORCRUX-DEV] Requesting threshold signature from peers: %v\n", s.peers)
	return []byte("horcrux_threshold_mock_signature_bytes_65_length_padded_out_here"), nil
}
