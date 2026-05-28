# Hugoverse 🚀

**Self-hosted Obsidian Sync & Publish Platform**

Hugoverse is an open-source, self-hosted alternative for Obsidian Sync and Publish. Deploy your own private sync server and publishing platform with automatic SSL, multi-user support, and enterprise-grade features.

## ✨ Features

- **🔄 Obsidian Sync**: Self-hosted CouchDB-based sync server
- **📝 Obsidian Publish**: Publish your notes to custom domains
- **🔐 License Management**: Built-in enterprise license system
- **🌐 Custom Domains**: Support for custom and wildcard domains
- **🔒 Auto SSL**: Automatic HTTPS with Let's Encrypt
- **📦 Docker**: One-command deployment
- **🎨 Modern UI**: Beautiful admin dashboard
- **👥 Multi-user**: Manage multiple users and licenses
- **💾 Automatic Backups**: Daily backups of CouchDB and sites
- **🔑 DNSPod Integration**: Wildcard SSL certificates support

## 🎯 Quick Start

### Free Version

The free version includes **1 enterprise license** at no cost. Perfect for personal use or trying out Hugoverse.

```bash
# Clone repository
git clone https://github.com/mdfriday/hugoverse.git
cd hugoverse

# Run interactive installation
bash install.sh
```

The script will guide you through:
1. Domain configuration
2. Admin credentials setup
3. Optional DNSPod configuration
4. Automatic service deployment

### Paid Version (Multiple Licenses)

Need more licenses for your team or business? Purchase a **Master License** to unlock higher quotas:

#### 📊 Pricing

| Plan | Sub-Licenses | Price | Best For |
|------|--------------|-------|----------|
| **Free** | 1 | $0 | Personal use |
| **Starter** | 10 | $99/year | Small teams |
| **Pro** | 100 | $499/year | Growing businesses |
| **Unlimited** | ∞ | $2,999 one-time | Enterprises |

Visit [mdfriday.com/pricing](https://mdfriday.com/pricing) to purchase.

#### Using Your Master License

```bash
# Option 1: During installation
# The install.sh script will prompt for your Master License

# Option 2: Add to existing deployment
nano .env
# Add: MASTER_LICENSE=YOUR_LICENSE_KEY

docker-compose restart hugoverse
```

Your Master License will be verified online, and you can generate licenses up to your quota.

## 📋 Prerequisites

- **Docker**: 20.10+ ([Install Docker](https://docs.docker.com/get-docker/))
- **Docker Compose**: 2.0+ (included in Docker Desktop)
- **Domain**: A domain with DNS access
- **Server**: Linux VPS with 2GB+ RAM (recommended)

## 🛠️ Manual Installation

If you prefer manual setup:

### 1. Clone Repository

```bash
git clone https://github.com/mdfriday/hugoverse.git
cd hugoverse
```

### 2. Configure Environment

```bash
cp .env.example .env
nano .env
```

Edit the following required fields:

```env
# Domain
DOMAIN=your-domain.com
SERVER_IP=your.server.ip

# Admin
ADMIN_EMAIL=admin@your-domain.com
ADMIN_PASSWORD=secure_password

# CouchDB
COUCHDB_PASSWORD=another_secure_password

# Master License (optional for free tier)
MASTER_LICENSE=
```

### 3. Start Services

```bash
docker-compose up -d
```

### 4. Check Status

```bash
# View logs
docker-compose logs -f hugoverse

# Check health
curl http://localhost:1314/api/health
```

## 🌐 DNS Configuration

Add these DNS records to your domain provider:

```
Type    Name    Value
A       @       YOUR_SERVER_IP
A       *       YOUR_SERVER_IP  (for wildcard domains)
```

Wait 5-30 minutes for DNS propagation.

## 📖 Usage

### Access Admin Panel

```
http://your-domain.com/admin
```

Login with your admin credentials.

### Generated Licenses

After installation, check the admin panel or logs for your generated enterprise license:

```bash
docker-compose logs hugoverse | grep "License Key"
```

You'll see output like:

```
License Key: MDF-XXXX-XXXX-XXXX
Email: mdf.XXXX.XXXX@mdfriday.com
Password: xxxxxxxxxx
```

Share these credentials with your users for activation in the Obsidian Friday plugin.

### Generate More Licenses (CLI)

If you have a Master License, you can generate licenses via CLI:

```bash
# Enter container
docker-compose exec hugoverse sh

# Generate licenses
/app/hugoverse license generate \
  -email admin@your-domain.com \
  -password your_password \
  -plan enterprise \
  -count 5
```

The system will verify your Master License quota before generation.

## 🔧 Configuration Options

### Environment Variables

All configuration is done via `.env` file. See `.env.example` for full options.

#### Core Settings

- `AUTO_INIT`: Auto-configure on first run (default: `true`)
- `DOMAIN`: Your domain name
- `SERVER_IP`: Server public IP for DNS verification
- `ADMIN_EMAIL`: Administrator email
- `ADMIN_PASSWORD`: Administrator password

#### CouchDB Settings

- `COUCHDB_USER`: CouchDB admin username (default: `admin`)
- `COUCHDB_PASSWORD`: CouchDB admin password (required)
- `COUCHDB_DB_PREFIX`: User database prefix (default: `userdb-`)

#### DNS Provider (Wildcard SSL via DNS-01)

The Caddy image bundles both Tencent Cloud DNS (DNSPod) and Alibaba Cloud DNS (AliDNS) plugins. Configure **one** provider whose nameservers actually serve your domain.

- `DNS_PROVIDER`: Explicit selector (recommended). Allowed values: `tencentcloud` | `alidns` | empty (legacy auto-detect)

Tencent Cloud DNS (DNSPod):

- `DNSPOD_ENABLED`: Enable DNSPod for wildcard certs (default: `false`)
- `DNSPOD_ID`: DNSPod Secret ID
- `DNSPOD_SECRET`: DNSPod Secret Key

Get DNSPod credentials: [console.dnspod.cn](https://console.dnspod.cn/account/token/apikey)

Alibaba Cloud DNS (AliDNS):

- `ALIDNS_ENABLED`: Enable AliDNS for wildcard certs (default: `false`)
- `ALIDNS_ACCESS_KEY_ID`: Aliyun AccessKey ID
- `ALIDNS_ACCESS_KEY_SECRET`: Aliyun AccessKey Secret

Get Aliyun credentials: [ram.console.aliyun.com/manage/ak](https://ram.console.aliyun.com/manage/ak)

> Backward compatibility: if `DNS_PROVIDER` is empty, the app falls back to `DNSPOD_ENABLED=true` → `tencentcloud`, matching pre-existing deployments.

#### Master License

- `MASTER_LICENSE`: Your purchased Master License key (optional)

Without this, you're limited to the free tier (1 license).

#### Enterprise Features

- `AUTO_GENERATE_ENTERPRISE_LICENSE`: Auto-generate license on startup (default: `true`)
- `ENTERPRISE_LICENSE_PLAN`: License plan (default: `enterprise`)
- `ENTERPRISE_LICENSE_COUNT`: Number to generate (default: `1`)

#### Backup

- `BACKUP_ENABLED`: Enable daily backups (default: `true`)
- `BACKUP_RETENTION_DAYS`: Days to keep backups (default: `7`)

## 📊 Service Management

### Start/Stop

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# Restart specific service
docker-compose restart hugoverse
```

### View Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f hugoverse
docker-compose logs -f caddy
docker-compose logs -f couchdb
```

### Update

```bash
# Pull latest images
docker-compose pull

# Restart services
docker-compose up -d
```

## 🔐 Security

### Best Practices

1. **Use strong passwords** for admin and CouchDB
2. **Enable HTTPS** with DNSPod for wildcard SSL
3. **Regular backups** are enabled by default
4. **Keep Master License secret** - never commit to Git
5. **Update regularly** to get security patches

### Firewall

Open these ports:

```bash
# Ubuntu/Debian
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# CentOS/RHEL
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --reload
```

## 🐛 Troubleshooting

### Services Not Starting

```bash
# Check Docker status
docker ps -a

# Check logs for errors
docker-compose logs hugoverse
```

### License Quota Exceeded

```
❌ License Quota Exceeded
   Current: 1 / 1 licenses
   Requested: 1
   Available: 0
```

**Solution**: Purchase a Master License from [mdfriday.com/pricing](https://mdfriday.com/pricing) and add to `.env`:

```env
MASTER_LICENSE=YOUR_LICENSE_KEY
```

Then restart:

```bash
docker-compose restart hugoverse
```

### DNS Not Resolving

```bash
# Check DNS propagation
dig your-domain.com
nslookup your-domain.com

# Verify A record points to your server IP
```

Wait up to 30 minutes for global DNS propagation.

### SSL Certificate Issues

If using DNSPod:

1. Verify DNSPod credentials are correct
2. Check DNS records are set
3. Review Caddy logs: `docker-compose logs caddy`

If using HTTP-01 (without DNSPod):

1. Ensure port 80 is accessible
2. Domain must point to your server
3. No firewall blocking HTTP

### Health Check Failing

```bash
# Check health endpoint
curl http://localhost:1314/api/health

# Should return:
# {"status":"healthy","docker":true,"initialized":true,"version":"latest"}
```

If unhealthy:
1. Check Hugoverse logs
2. Verify CouchDB is running
3. Verify Caddy Admin API is accessible

## 🔄 Backup & Restore

### Manual Backup

```bash
# Backup CouchDB data
docker-compose exec couchdb couchbackup --db DBNAME > backup.json

# Backup volumes
docker run --rm -v hugoverse_couchdb_data:/data -v $(pwd):/backup \
  alpine tar czf /backup/couchdb-backup.tar.gz /data
```

### Automatic Backups

Automatic backups run daily and are stored in `/backups` volume. Retention is configurable via `BACKUP_RETENTION_DAYS`.

### Restore

```bash
# Stop services
docker-compose down

# Restore volume
docker run --rm -v hugoverse_couchdb_data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/couchdb-backup.tar.gz -C /

# Start services
docker-compose up -d
```

## 📚 Documentation

- **GitHub**: [github.com/mdfriday/hugoverse](https://github.com/mdfriday/hugoverse)
- **Issues**: [github.com/mdfriday/hugoverse/issues](https://github.com/mdfriday/hugoverse/issues)
- **Docs**: [docs.mdfriday.com](https://docs.mdfriday.com)

## 💬 Support

- **Email**: support@mdfriday.com
- **GitHub Issues**: For bug reports and feature requests
- **Community**: Join our Discord (coming soon)

## 📄 License

Hugoverse source code: Closed source
Docker images: Public (see LICENSE file)

Master License required for generating multiple sub-licenses.

## 🙏 Acknowledgments

- [Obsidian](https://obsidian.md) - The amazing note-taking app
- [CouchDB](https://couchdb.apache.org/) - Reliable sync backend
- [Caddy](https://caddyserver.com/) - Automatic HTTPS
- Community contributors

---

**Made with ❤️ by [MDFriday](https://mdfriday.com)**

Need more licenses? [Visit our pricing page](https://mdfriday.com/pricing)
