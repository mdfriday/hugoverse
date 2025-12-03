# MDFriday License System Usage Guide

## Overview

The MDFriday License System provides three types of licenses:
- **Free**: Basic functionality with branding (30 days)
- **Yearly**: Full features with server-side services, expires after 1 year
- **Lifetime**: Full features without server dependency, never expires (9999-01-01)

Each license supports up to **3 device activations** by default.

## CLI Commands

### 1. Generate Cryptographic Keys

First, generate the cryptographic keys needed for the license system:

```bash
hugov license keygen
```

This creates the following files in `~/.mdfriday/keys/`:
- `ecdsa_private.pem` - Used for signing licenses
- `ecdsa_public.pem` - Used by frontend to verify signatures
- `rsa_private.pem` - Used for encrypting content keys
- `rsa_public.pem` - Used by frontend to decrypt content keys

License files are stored in `~/.mdfriday/licenses/`

### 2. Generate License Keys

Generate multiple license keys with the new format:

```bash
# Generate 5 lifetime licenses
hugov license generate -plan lifetime -count 5

# Generate 10 yearly licenses (saved to ~/.mdfriday/licenses/)
hugov license generate -plan yearly -count 10

# Generate free trial licenses
hugov license generate -plan free -count 3

# Generate to custom directory
hugov license generate -plan lifetime -count 5 -output-dir /custom/path
```

This creates files in `~/.mdfriday/licenses/`:
- Individual JSON files for each license key: `MDF-XXXX-XXXX-XXXX.json`
- A registry file: `licenses_[plan]_[date].json`

### 3. Activate a License

Activate a license key with device binding:

```bash
hugov license activate -key MDF-ABCD-EFGH-JKLM -device-id my-device-123
```

### 4. Verify a License

Verify that a license is valid:

```bash
hugov license verify -license ~/.mdfriday/licenses/MDF-ABCD-EFGH-JKLM_lifetime.mdf.license
```

## File Structure

### Registry File Format
```json
{
  "generatedAt": "2025-12-03",
  "plan": "lifetime",
  "count": 5,
  "licenseKeys": [
    "MDF-ABCD-EFGH-JKLM",
    "MDF-NOPQ-RSTU-VWXY"
  ]
}
```

### Individual License Detail Format
```json
{
  "licenseKey": "MDF-ABCD-EFGH-JKLM",
  "plan": "lifetime",
  "issueDate": "2025-12-03",
  "expiryDate": "9999-01-01",
  "maxActivations": 3,
  "currentActivations": 1,
  "deviceIds": ["device-123"],
  "resourceKey": "encrypted-cek-data",
  "version": 1
}
```

## API Endpoints

### 1. Activate License
```http
POST /api/license/activate
Content-Type: application/json

{
  "licenseKey": "MDF-ABCD-EFGH-JKLM",
  "deviceId": "my-device-123"
}
```

Response:
```json
{
  "success": true,
  "license": {
    "payload": "base64-encoded-license-data",
    "signature": "ecdsa-signature"
  },
  "publicKeys": {
    "ecdsaPublicKey": "-----BEGIN PUBLIC KEY-----...",
    "rsaPublicKey": "-----BEGIN PUBLIC KEY-----..."
  },
  "detail": {
    "licenseKey": "MDF-ABCD-EFGH-JKLM",
    "plan": "lifetime",
    "currentActivations": 1,
    "maxActivations": 3
  }
}
```

### 2. Get Public Keys
```http
GET /api/license/public-keys
```

### 3. Validate License Key
```http
POST /api/license/validate
Content-Type: application/json

{
  "licenseKey": "MDF-ABCD-EFGH-JKLM"
}
```

## Security Features

1. **ECDSA Signature**: Prevents license forgery
2. **RSA Encryption**: Protects content encryption keys
3. **AES-GCM**: Encrypts premium theme content
4. **Device Binding**: Each activation is tied to a specific device
5. **Multi-Device Support**: Up to 3 devices per license
6. **Offline Verification**: No server dependency for activated licenses

## Date Format

All dates use the format `YYYY-MM-DD`:
- Issue dates: Current date when generated
- Expiry dates: 
  - Free: 30 days from issue
  - Yearly: 1 year from issue  
  - Lifetime: `9999-01-01` (never expires)

## Integration Notes

- Frontend needs both public keys to verify and decrypt
- License keys use format: `MDF-XXXX-XXXX-XXXX`
- Each license supports up to 3 device activations
- Activation API returns complete license payload for frontend use
- All API endpoints are public (no authentication required)
