# Local Signature Aggregator

A temporary experimental tool for collecting and aggregating signatures for subnet to L1 conversions on Avalanche networks, specifically designed for nodes behind NAT or without dedicated IP addresses.

## Overview

This tool helps collect conversion signatures when your validator node is behind a NAT or firewall and doesn't have a dedicated IP address accessible to other validators, which is common for most laptops and home setups.

## Docker Usage

You can pull the Docker image from Docker Hub:
```
docker pull containerman17/local_agg
```

Run using Docker:
```
docker run -it --net=host --rm containerman17/local_agg YOUR_L1ID
```

Supports both amd64 and arm64 architectures.

## Purpose

This tool was created as an alternative to the [Avalanche L1 Toolbox's collectConversionSignatures](https://build.avax.network/tools/l1-toolbox#collectConversionSignatures) functionality for nodes that cannot be directly reached by other validators.

When successful, it outputs a signature that can be pasted into the L1 toolbox.

## Note

This is an experimental tool intended for temporary use when standard methods aren't feasible due to network configuration constraints.
