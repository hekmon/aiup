# gencatalog

GPU catalog generator for MSI Afterburner profiles.

## Overview

This tool automatically generates the GPU and manufacturer catalog from the [pci-ids](https://pci-ids.ucw.cz/) database, filtering for GPU vendors supported by MSI Afterburner (NVIDIA, AMD, Intel).

## Purpose

Instead of manually maintaining the GPU catalog in `catalog.go`, this tool:

- Fetches the latest pci-ids database
- Extracts only GPU/display devices from supported vendors
- Cleans up device names to be user-friendly
- Generates Go code that can be used by the profiles package

## Usage

### Generate the catalog

From the project root directory, run:

```bash
go generate ./profiles/...
```

This will:
1. Download the latest pci-ids database from https://pci-ids.ucw.cz/v2.2/pci.ids
2. Parse and filter for NVIDIA, AMD, and Intel GPUs
3. Generate `profiles/catalog_generated.go` with the catalog data

### Manual execution

You can also run the generator directly:

```bash
go run ./cmd/gencatalog/main.go
```

## Output

The generator produces `catalog_generated.go` containing:

- `gpuCatalog`: Map of "VendorID_DeviceID" to GPU names
- `manufacturerCodes`: Map of subsystem vendor codes to manufacturer names

Example entries:
```go
"10de_2b85": {Vendor: "NVIDIA", GPU: "GeForce RTX 5090"}
"1002_73af": {Vendor: "AMD", GPU: "Radeon RX 6900 XT"}
"8086_56a0": {Vendor: "Intel", GPU: "Arc A770"}
```

## Supported Manufacturers

The generator includes manufacturer codes for MSI Afterburner-supported brands:
- ASUS, MSI, NVIDIA (Founders Edition), EVGA, Gigabyte
- Gainward, Zotac, Sapphire, PowerColor, XFX
- PNY, Palit, Colorful, Galax/KFA2
- Dell, HP, Lenovo

## Notes

- The generated file should **NOT** be edited manually
- Device IDs are in lowercase (pci-ids format)
- The lookup functions automatically normalize input to lowercase
- Non-GPU devices (audio, USB, network, etc.) are filtered out

## Troubleshooting

If you encounter issues:

1. **Network errors**: Ensure you have internet access to fetch pci-ids
2. **Build errors**: Run `go generate` before building
3. **Outdated catalog**: Re-run `go generate` to fetch the latest database

## Development

To modify the filtering or name formatting logic, edit `cmd/gencatalog/main.go` and re-run the generator.