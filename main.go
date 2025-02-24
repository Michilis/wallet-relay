package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fiatjaf/eventstore/lmdb"
	"github.com/fiatjaf/khatru"
	"github.com/joho/godotenv"
	"github.com/nbd-wtf/go-nostr"
)

var config Config
var relay *khatru.Relay
var walletKinds = []int{
	nostr.KindNWCWalletInfo,
	nostr.KindNWCWalletRequest,
	nostr.KindNWCWalletResponse,
	nostr.KindNutZap,
	nostr.KindNutZapInfo,
	nostr.KindZap,
	nostr.KindZapRequest,
	17375,
	7375,
	7376,
	7374,
	// NIP-87 kinds
	38000, // Recommendation Event
	38172, // Cashu Mint Announcement
	38173, // Fedimint Announcement,
	// NIP-61 kinds
	10019, // Nutzap Informational Event
	9321,  // Nutzap Event
	23194, // Wallet Requests (Optional) - NIP-47
	31990, // Data Vending Machine (Optional) - NIP-90
	3,     // OpenTimestamps (NIP-3)
	5,     // Event Deletion (NIP-9)
}

type Config struct {
	RelayName        string
	RelayPubkey      string
	RelayDescription string
	RelayIcon        string
	RelaySoftware    string
	RelayVersion     string
	RelayPort        string
	LmdbMapSize      int64
	LmdbPath         string
}

func main() {
	relay = khatru.NewRelay()
	config = LoadConfig()

	db := lmdb.LMDBBackend{
		Path:    config.LmdbPath,
		MapSize: config.LmdbMapSize,
	}

	if err := db.Init(); err != nil {
		panic(err)
	}

	// Custom authentication logic
	relay.RejectEvent = append(relay.RejectEvent, func(ctx context.Context, event *nostr.Event) (bool, string) {
		if !verifyEventSignature(event) {
			return true, "invalid-signature: authentication failed"
		}
		if !containsOnlyWalletKids([]int{event.Kind}) {
			fmt.Println(MsgPublishFail, event.Kind)
			return true, MsgInvalidEvent
		}

		return false, ""
	})

	relay.StoreEvent = append(relay.StoreEvent, db.SaveEvent)
	relay.QueryEvents = append(relay.QueryEvents, db.QueryEvents)
	relay.ReplaceEvent = append(relay.ReplaceEvent, db.ReplaceEvent)

	addr := fmt.Sprintf("%s:%s", "0.0.0.0", config.RelayPort)

	log.Printf(MsgListening, addr)
	err := http.ListenAndServe(addr, relay)
	if err != nil {
		log.Fatal(err)
	}
}

func LoadConfig() Config {
	_ = godotenv.Load(".env")

	config = Config{
		RelayName:        os.Getenv("RELAY_NAME"),
		RelayPubkey:      os.Getenv("RELAY_PUBKEY"),
		RelayDescription: os.Getenv("RELAY_DESCRIPTION"),
		RelayIcon:        os.Getenv("RELAY_ICON"),
		RelaySoftware:    "https://github.com/bitvora/wallet-relay",
		RelayVersion:     "0.1.0",
		RelayPort:        os.Getenv("RELAY_PORT"),
		LmdbPath:         os.Getenv("LMDB_PATH"),
	}

	relay.Info.Name = config.RelayName
	relay.Info.PubKey = config.RelayPubkey
	relay.Info.Description = config.RelayDescription
	relay.Info.Icon = config.RelayIcon
	relay.Info.Software = config.RelaySoftware
	relay.Info.Version = config.RelayVersion

	return config
}

func containsOnlyWalletKids(kinds []int) bool {
	walletKindSet := make(map[int]struct{})

	for _, walletKind := range walletKinds {
		walletKindSet[walletKind] = struct{}{}
	}
	for _, kind := range kinds {
		if _, exists := walletKindSet[kind]; !exists {
			return false
		}
	}

	return true
}

// Custom function to verify event signatures
func verifyEventSignature(event *nostr.Event) bool {
	// Implement signature verification logic here
	// This is a placeholder and should be replaced with actual verification code
	return true
}
