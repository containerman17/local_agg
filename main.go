package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/netip"
	"os"
	"time"

	"github.com/ava-labs/avalanchego/api/info"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/message"
	"github.com/ava-labs/avalanchego/network/peer"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/platformvm"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	warpMessage "github.com/ava-labs/avalanchego/vms/platformvm/warp/message"
	warpPayload "github.com/ava-labs/avalanchego/vms/platformvm/warp/payload"
	"github.com/ava-labs/icm-services/config"
	"github.com/ava-labs/icm-services/peers"
	"github.com/ava-labs/icm-services/signature-aggregator/aggregator"
	sigAggConfig "github.com/ava-labs/icm-services/signature-aggregator/config"
	"github.com/ava-labs/icm-services/signature-aggregator/metrics"
	"github.com/ava-labs/icm-services/types"
	"github.com/prometheus/client_golang/prometheus"
)

const defaultEndpoint = "https://api.avax-test.network"

func main() {
	// Parse command line flags
	host := flag.String("host", "127.0.0.1", "Host IP address")
	port := flag.Int("port", 9651, "Port number")
	flag.Parse()

	// Get the required positional argument (conversionID)
	args := flag.Args()
	if len(args) < 1 {
		log.Fatalf("Error: conversionID is required")
	}
	conversionID := args[0]

	// Setup logger
	logger, err := setupLogger()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}

	// Generate peers list
	peersList := generatePeers(*host, *port)

	// Get message from transaction
	decodedMessage, subnetID, err := getMessage(conversionID)
	if err != nil {
		log.Fatalf("failed to get message from tx: %v", err)
	}

	// Create network and sign message
	err = processWarpMessage(defaultEndpoint, logger, peersList, subnetID, decodedMessage)
	if err != nil {
		log.Fatalf("failed to process warp message: %v", err)
	}
}

func setupLogger() (logging.Logger, error) {
	return logging.NewFactory(logging.Config{
		LogLevel:     logging.Fatal,
		DisplayLevel: logging.Warn,
	}).Make("hello")
}

func generatePeers(host string, port int) []info.Peer {
	peers := []info.Peer{
		{
			Info: peer.Info{
				IP:       netip.AddrPortFrom(netip.MustParseAddr(host), uint16(port)),
				PublicIP: netip.AddrPortFrom(netip.MustParseAddr(host), uint16(port)),
			},
			Benched: []string{},
		},
	}

	if port != 9651 {
		// Add 9651
		peers = append(peers, info.Peer{
			Info: peer.Info{
				IP:       netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), 9651),
				PublicIP: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), 9651),
			},
			Benched: []string{},
		})

		// Add port+1 just in case if user confused it with http port
		peers = append(peers, info.Peer{
			Info: peer.Info{
				IP:       netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(port+1)),
				PublicIP: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(port+1)),
			},
			Benched: []string{},
		})
	}

	return peers
}

func processWarpMessage(endpoint string, logger logging.Logger, peersList []info.Peer, subnetID ids.ID, decodedMessage []byte) error {
	peerNetwork, err := createAppRequestNetwork(
		endpoint,
		logger,
		peersList,
		subnetID.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to create peer network: %w", err)
	}

	messageCreator, err := message.NewCreator(
		logger,
		prometheus.NewRegistry(),
		constants.DefaultNetworkCompressionType,
		constants.DefaultNetworkMaximumInboundTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to create message creator: %w", err)
	}

	signatureAggregator, err := aggregator.NewSignatureAggregator(
		peerNetwork,
		logger,
		messageCreator,
		uint64(1024*1024),
		metrics.NewSignatureAggregatorMetrics(prometheus.NewRegistry()),
	)
	if err != nil {
		return fmt.Errorf("failed to create signature aggregator: %w", err)
	}

	warpMsg, err := types.UnpackWarpMessage(decodedMessage)
	if err != nil {
		return fmt.Errorf("failed to unpack warp message: %w", err)
	}

	for i := 0; i < 100; i++ {
		signed, err := signatureAggregator.CreateSignedMessage(
			warpMsg,
			subnetID[:],
			subnetID,
			uint64(67),
		)
		if err == nil {
			fmt.Printf("Here is your signature, paste it into the toolbox: \n%v\n", "0x"+hex.EncodeToString(signed.Bytes()))
			os.Exit(0)
		}

		log.Printf("attempt %d failed: %v", i+1, err)
	}

	return fmt.Errorf("all attempts to sign message failed")
}

// getMessage retrieves and processes a transaction by ID
func getMessage(txId string) ([]byte, ids.ID, error) {
	txID, err := ids.FromString(txId)
	if err != nil {
		return nil, ids.ID{}, fmt.Errorf("failed to parse tx id: %w", err)
	}
	endpoint := "https://api.avax-test.network" // Fuji endpoint
	pClient := platformvm.NewClient(endpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	txBytes, err := pClient.GetTx(ctx, txID)
	if err != nil {
		return nil, ids.ID{}, fmt.Errorf("failed to get transaction bytes: %w", err)
	}

	var tx txs.Tx
	if _, err := txs.Codec.Unmarshal(txBytes, &tx); err != nil {
		return nil, ids.ID{}, fmt.Errorf("failed to unmarshal tx: %w", err)
	}

	if err := tx.Initialize(txs.Codec); err != nil {
		return nil, ids.ID{}, fmt.Errorf("failed to initialize tx: %w", err)
	}

	convertTx, ok := tx.Unsigned.(*txs.ConvertSubnetToL1Tx)
	if !ok {
		return nil, ids.ID{}, fmt.Errorf("unexpected transaction type: %T", tx.Unsigned)
	}

	validators := []warpMessage.SubnetToL1ConversionValidatorData{}
	for _, validator := range convertTx.Validators {
		validators = append(validators, warpMessage.SubnetToL1ConversionValidatorData{
			NodeID:       validator.NodeID,
			BLSPublicKey: validator.Signer.PublicKey,
			Weight:       validator.Weight,
		})
	}

	subnetToL1ConversionData := warpMessage.SubnetToL1ConversionData{
		SubnetID:       convertTx.Subnet,
		ManagerChainID: convertTx.ChainID,
		ManagerAddress: convertTx.Address,
		Validators:     validators,
	}
	subnetToL1ConversionID, err := warpMessage.SubnetToL1ConversionID(subnetToL1ConversionData)
	if err != nil {
		return nil, ids.ID{}, fmt.Errorf("failed to create subnetToL1ConversionID: %w", err)
	}

	subnetToL1ConversionPayload, err := warpMessage.NewSubnetToL1Conversion(subnetToL1ConversionID)
	if err != nil {
		return nil, ids.ID{}, fmt.Errorf("could not create subnetToL1ConversionPayload: %v", err)
	}

	addressedCall, err := warpPayload.NewAddressedCall(
		nil, //manager address, nil since talking to p-chain
		subnetToL1ConversionPayload.Bytes(),
	)
	if err != nil {
		return nil, ids.ID{}, fmt.Errorf("could not generate addressed call: %v", err)
	}
	unsignedMessage, err := warp.NewUnsignedMessage(
		constants.FujiID,          //  Network ID
		constants.PlatformChainID, //  Blockchain ID (P-Chain ID)
		addressedCall.Bytes(),     // The payload
	)
	if err != nil {
		return nil, ids.ID{}, fmt.Errorf("failed to create unsigned warp message: %v", err)
	}

	return unsignedMessage.Bytes(), convertTx.Subnet, nil
}

// createAppRequestNetwork creates a network for application requests
func createAppRequestNetwork(
	endpoint string,
	logger logging.Logger,
	extraPeerEndpoints []info.Peer,
	subnetId string,
) (peers.AppRequestNetwork, error) {
	networkConfig := sigAggConfig.Config{
		PChainAPI: &config.APIConfig{
			BaseURL: endpoint,
		},
		InfoAPI: &config.APIConfig{
			BaseURL: endpoint,
		},
		AllowPrivateIPs:  true,
		TrackedSubnetIDs: []string{subnetId},
	}
	if err := networkConfig.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate peer network config: %w", err)
	}
	peerNetwork, err := peers.NewNetwork(
		logger,
		prometheus.NewRegistry(),
		networkConfig.GetTrackedSubnets(),
		extraPeerEndpoints,
		&networkConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer network: %w", err)
	}
	return peerNetwork, nil
}
