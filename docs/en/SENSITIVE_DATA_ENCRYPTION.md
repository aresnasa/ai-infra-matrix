# Sensitive Data Encryption Documentation

**[中文文档](../zh_CN/SENSITIVE_DATA_ENCRYPTION.md)** | **English**

## Overview

ai-infra-matrix v0.3.8+ introduces database sensitive data encryption functionality to protect the following sensitive information stored in the database:

- SSH passwords and usernames (SLURM node configuration)
- SSH private key paths (SLURM node configuration)
- AI assistant API keys and secrets (AI configuration)

## Encryption Algorithm

- Uses **AES-256-GCM** symmetric encryption algorithm
- Key derived from environment variable `ENCRYPTION_KEY` via **SHA-256**
- Random **Nonce** used for each encryption, ensuring identical plaintext produces different ciphertext
- Encrypted data prefixed with `encrypted:` for identification

## Configuration

### 1. Set Encryption Key

Configure encryption key in `.env` file:

```bash
# Encryption configuration (for sensitive data encryption, such as SSH passwords, API keys, etc.)
ENCRYPTION_KEY=your-encryption-key-change-in-production-32-bytes
```

**Important Notes:**
- Production environment must use a strong random key
- Recommended key length is at least 32 characters
- Once set, do not change the key arbitrarily, otherwise encrypted data cannot be decrypted
- Recommended command to generate a secure key:
  ```bash
  openssl rand -base64 32
  ```

### 2. Docker Compose Configuration

In `docker-compose.yml.tpl`, backend service is auto-configured:

```yaml
environment:
  - ENCRYPTION_KEY=${ENCRYPTION_KEY:-your-encryption-key-change-in-production-32-bytes}
```

## How It Works

### Automatic Encryption/Decryption

Transparent encryption implemented via GORM Hooks:

1. **BeforeCreate/BeforeUpdate/BeforeSave**: Auto-encrypt sensitive fields before storage
2. **AfterFind**: Auto-decrypt after reading from database

### Protected Models

#### SlurmNode
| Field | Description |
|-------|-------------|
| Username | SSH username (encrypted storage) |
| Password | SSH password (encrypted storage) |
| KeyPath | SSH private key path (not exposed in JSON) |

#### SlurmCluster
| Field | Description |
|-------|-------------|
| MasterSSH.Username | Master node SSH username (encrypted storage) |
| MasterSSH.Password | Master node SSH password (encrypted storage) |

#### AIAssistantConfig
| Field | Description |
|-------|-------------|
| APIKey | API key (encrypted storage) |
| APISecret | API secret (encrypted storage) |

### API Response Security

Sensitive fields are hidden during JSON serialization (`json:"-"`):
- Frontend cannot directly access password/key values
- For AI configuration, `has_api_key` and `has_api_secret` boolean fields indicate whether configured

## Migrating Existing Data

If upgrading to v0.3.8+, existing plaintext data needs migration:

```bash
# Compile migration tool
cd src/backend
go build -o migrate-encryption ./cmd/migrate-encryption/main.go

# Run migration
./migrate-encryption
```

The migration tool will:
1. Check all sensitive fields
2. Identify unencrypted plaintext data
3. Auto-encrypt and update database
4. Output migration report

## Security Best Practices

1. **Key Management**
   - Use environment variables or key management service to store `ENCRYPTION_KEY`
   - Do not hardcode keys in code or configuration files
   - Rotate keys periodically (requires re-encrypting all data)

2. **Backup**
   - Backup `ENCRYPTION_KEY`, loss will result in inability to decrypt data
   - Database backups and keys should be stored separately

3. **Transport Security**
   - Always use HTTPS for sensitive data transmission
   - Internal service communication should also be encrypted

4. **Log Auditing**
   - Sensitive fields are not output in logs
   - Enable access logs to record abnormal access

## Troubleshooting

### Decryption Failed

If you encounter "failed to decrypt" error:

1. Check if `ENCRYPTION_KEY` matches the key used during encryption
2. Confirm encryption service is properly initialized
3. Check if data is corrupted

### Plaintext Data Not Encrypted

If database still has plaintext:

1. Confirm `ENCRYPTION_KEY` is properly configured
2. Run migration tool
3. Check if GORM Hooks are working properly

## Technical Details

### Encrypted Data Format

```
encrypted:BASE64(nonce + ciphertext + tag)
```

- `nonce`: 12-byte random number
- `ciphertext`: Encrypted data
- `tag`: GCM authentication tag

### Backward Compatibility

- Unencrypted data can be read (`DecryptSafely` returns original text)
- System auto-detects `encrypted:` prefix to identify encrypted data
- Unencrypted data is auto-encrypted on save

## Related Files

- `src/backend/internal/utils/encryption.go` - Encryption service implementation
- `src/backend/internal/utils/encryption_manager.go` - Global encryption manager
- `src/backend/internal/models/slurm_cluster_models.go` - SLURM model encryption Hooks
- `src/backend/internal/models/ai_assistant.go` - AI config encryption Hooks
- `src/backend/cmd/migrate-encryption/main.go` - Data migration tool
