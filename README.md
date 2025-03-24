# Local Signature Aggregator

A temporary experimental tool for collecting and aggregating signatures for subnet to L1 conversions on Avalanche networks, specifically designed for nodes behind NAT or without dedicated IP addresses.

## Overview

This tool helps collect conversion signatures when your validator node is behind a NAT or firewall and doesn't have a dedicated IP address accessible to other validators, which is common for most laptops and home setups.

## Usage

```
./local_agg [flags] <conversionID>
```

Flags:
- `-host`: Host IP address (default "127.0.0.1")
- `-port`: Port number (default 9651)

Example:
```
./local_agg -host 127.0.0.1 -port 9651 2sEjTD89o5VRD8FZUKC2PzKMwEw1Sh9Wi45hbvjJcGJijr4Srz
```

## Docker

You can pull the Docker image from Docker Hub:
```
docker pull containerman17/local_agg:latest
```

Run using Docker:
```
docker run -it containerman17/local_agg -host <your_ip> -port <your_port> <conversionID>
```

Supports both amd64 and arm64 architectures.

## Purpose

This tool was created as an alternative to the [Avalanche L1 Toolbox's collectConversionSignatures](https://build.avax.network/tools/l1-toolbox#collectConversionSignatures) functionality for nodes that cannot be directly reached by other validators.

When successful, it outputs a signature that can be pasted into the L1 toolbox.

## Note

This is an experimental tool intended for temporary use when standard methods aren't feasible due to network configuration constraints.
