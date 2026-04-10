#!/bin/bash
set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
cat << "EOF"
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║             🚀 Hugoverse Installation Script            ║
║                                                          ║
║     Self-hosted Obsidian Sync & Publish Platform        ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"
echo ""

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  Warning: Running as root is not recommended${NC}"
    echo -e "   Please run this script as a regular user with Docker permissions"
    echo ""
fi

# Check Docker
echo -e "${BLUE}📦 Checking prerequisites...${NC}"
echo ""

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    echo ""
    echo "Please install Docker first:"
    echo "  Ubuntu/Debian: curl -fsSL https://get.docker.com | sh"
    echo "  macOS: https://docs.docker.com/desktop/install/mac-install/"
    echo "  Windows: https://docs.docker.com/desktop/install/windows-install/"
    exit 1
fi
echo -e "${GREEN}✅ Docker found: $(docker --version)${NC}"

# Check Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed${NC}"
    echo ""
    echo "Please install Docker Compose:"
    echo "  https://docs.docker.com/compose/install/"
    exit 1
fi

# Detect compose command
if docker compose version &> /dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
else
    COMPOSE_CMD="docker-compose"
fi
echo -e "${GREEN}✅ Docker Compose found${NC}"
echo ""

# Interactive Configuration
echo -e "${BLUE}⚙️  Configuration${NC}"
echo ""
echo "Please provide the following information:"
echo ""

# Domain
read -p "$(echo -e ${BLUE}📍 Domain name${NC}) (e.g., example.com): " DOMAIN
while [ -z "$DOMAIN" ]; do
    echo -e "${RED}Domain cannot be empty${NC}"
    read -p "$(echo -e ${BLUE}📍 Domain name${NC}) (e.g., example.com): " DOMAIN
done

# Server IP
read -p "$(echo -e ${BLUE}🌐 Server public IP${NC}) (e.g., 1.2.3.4): " SERVER_IP
while [ -z "$SERVER_IP" ]; do
    echo -e "${RED}Server IP cannot be empty${NC}"
    read -p "$(echo -e ${BLUE}🌐 Server public IP${NC}) (e.g., 1.2.3.4): " SERVER_IP
done

# Admin Email
read -p "$(echo -e ${BLUE}👤 Admin email${NC}): " ADMIN_EMAIL
while [ -z "$ADMIN_EMAIL" ]; do
    echo -e "${RED}Admin email cannot be empty${NC}"
    read -p "$(echo -e ${BLUE}👤 Admin email${NC}): " ADMIN_EMAIL
done

# Admin Password
read -s -p "$(echo -e ${BLUE}🔒 Admin password${NC}) (min 6 chars): " ADMIN_PASSWORD
echo ""
while [ ${#ADMIN_PASSWORD} -lt 6 ]; do
    echo -e "${RED}Password must be at least 6 characters${NC}"
    read -s -p "$(echo -e ${BLUE}🔒 Admin password${NC}) (min 6 chars): " ADMIN_PASSWORD
    echo ""
done

# CouchDB Password
read -s -p "$(echo -e ${BLUE}🗄️  CouchDB password${NC}) (min 6 chars): " COUCHDB_PASSWORD
echo ""
while [ ${#COUCHDB_PASSWORD} -lt 6 ]; do
    echo -e "${RED}Password must be at least 6 characters${NC}"
    read -s -p "$(echo -e ${BLUE}🗄️  CouchDB password${NC}) (min 6 chars): " COUCHDB_PASSWORD
    echo ""
done

# DNSPod Configuration
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}🌐 DNSPod Configuration (Optional)${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "DNSPod enables wildcard SSL certificates (*.${DOMAIN})"
echo "If you don't have DNSPod credentials, press Enter to skip."
echo ""
echo "Get credentials at: https://console.dnspod.cn/account/token/apikey"
echo ""

read -p "$(echo -e ${BLUE}🔑 DNSPod ID${NC}) (optional): " DNSPOD_ID
read -p "$(echo -e ${BLUE}🔐 DNSPod Secret${NC}) (optional): " DNSPOD_SECRET

if [ -n "$DNSPOD_ID" ] && [ -n "$DNSPOD_SECRET" ]; then
    DNSPOD_ENABLED="true"
    echo -e "${GREEN}✅ DNSPod will be enabled${NC}"
else
    DNSPOD_ENABLED="false"
    echo -e "${YELLOW}⚠️  DNSPod disabled - using HTTP-01 validation${NC}"
fi

# Master License
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}🔑 Master License (Optional)${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Free version includes 1 enterprise license."
echo "Purchase Master License for more quotas at:"
echo "  https://mdfriday.com/pricing"
echo ""
read -p "$(echo -e ${BLUE}📜 Master License${NC}) (optional): " MASTER_LICENSE

if [ -z "$MASTER_LICENSE" ]; then
    echo -e "${GREEN}✅ Using FREE mode (1 enterprise license)${NC}"
else
    echo -e "${GREEN}✅ Master License will be verified online${NC}"
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📝 Configuration Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  Domain: $DOMAIN"
echo "  Server IP: $SERVER_IP"
echo "  Admin Email: $ADMIN_EMAIL"
echo "  DNSPod: $DNSPOD_ENABLED"
if [ -n "$MASTER_LICENSE" ]; then
    echo "  Master License: Provided"
else
    echo "  Master License: FREE (1 license)"
fi
echo ""

read -p "$(echo -e ${YELLOW}Continue with installation?${NC}) [Y/n]: " CONFIRM
CONFIRM=${CONFIRM:-Y}
if [[ ! $CONFIRM =~ ^[Yy]$ ]]; then
    echo -e "${RED}Installation cancelled${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}🚀 Starting Installation${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Generate .env file
echo -e "${BLUE}📝 Generating .env file...${NC}"
cat > .env << EOF
# Hugoverse Configuration
# Generated on $(date)

# ---- Auto Initialization ----
AUTO_INIT=true

# ---- Domain Configuration ----
DOMAIN=$DOMAIN
SERVER_IP=$SERVER_IP
SITE_NAME=Hugoverse

# ---- Administrator Configuration ----
ADMIN_EMAIL=$ADMIN_EMAIL
ADMIN_PASSWORD=$ADMIN_PASSWORD

# ---- CouchDB Configuration ----
COUCHDB_USER=admin
COUCHDB_PASSWORD=$COUCHDB_PASSWORD
COUCHDB_DB_PREFIX=userdb-

# ---- DNSPod Configuration ----
DNSPOD_ENABLED=$DNSPOD_ENABLED
DNSPOD_ID=$DNSPOD_ID
DNSPOD_SECRET=$DNSPOD_SECRET

# ---- Master License ----
MASTER_LICENSE=$MASTER_LICENSE

# ---- Enterprise License Auto-Generation ----
AUTO_GENERATE_ENTERPRISE_LICENSE=true
ENTERPRISE_LICENSE_PLAN=enterprise
ENTERPRISE_LICENSE_COUNT=1

# ---- Enterprise Site Auto-Configuration ----
AUTO_CONFIGURE_ENTERPRISE_SITE=true
ENTERPRISE_SITE_DOMAIN=$DOMAIN

# ---- Backup Configuration ----
BACKUP_ENABLED=true
BACKUP_RETENTION_DAYS=7

# ---- Advanced Configuration ----
VERSION=latest
HTTP_PORT=80
HTTPS_PORT=443
LOG_LEVEL=info
EOF

echo -e "${GREEN}✅ .env file created${NC}"

# Create data directories
echo ""
echo -e "${BLUE}📁 Creating data directories...${NC}"
mkdir -p data/hugoverse data/couchdb data/caddy data/backups
echo -e "${GREEN}✅ Data directories created${NC}"

# Pull images
echo ""
echo -e "${BLUE}📦 Pulling Docker images...${NC}"
$COMPOSE_CMD pull
echo -e "${GREEN}✅ Images pulled${NC}"

# Start services
echo ""
echo -e "${BLUE}🚀 Starting services...${NC}"
$COMPOSE_CMD up -d

echo ""
echo -e "${BLUE}⏳ Waiting for services to be ready...${NC}"
sleep 10

# Check health
MAX_RETRIES=30
RETRY_COUNT=0
echo -e "${BLUE}🔍 Checking Hugoverse health...${NC}"

while [ $RETRY_COUNT -lt $MAX_RETRIES ]; do
    if curl -sf http://localhost:1314/api/health > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Hugoverse is healthy!${NC}"
        break
    fi
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $((RETRY_COUNT % 5)) -eq 0 ]; then
        echo -e "   Still waiting... ($RETRY_COUNT/$MAX_RETRIES)"
    fi
    sleep 2
done

if [ $RETRY_COUNT -eq $MAX_RETRIES ]; then
    echo -e "${RED}❌ Hugoverse health check failed${NC}"
    echo ""
    echo "Check logs with:"
    echo "  $COMPOSE_CMD logs -f hugoverse"
    exit 1
fi

# Installation complete
echo ""
echo -e "${GREEN}"
cat << "EOF"
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║        ✅ Hugoverse Installed Successfully! 🎉          ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"
echo ""

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📍 Access Information${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  Admin Panel: ${GREEN}http://${DOMAIN}/admin${NC}"
echo -e "  Login Email: ${GREEN}${ADMIN_EMAIL}${NC}"
echo ""
if [ "$DNSPOD_ENABLED" = "true" ]; then
    echo -e "  CouchDB: ${GREEN}https://cdb.${DOMAIN}${NC}"
else
    echo -e "  CouchDB: ${GREEN}http://cdb.${DOMAIN}${NC}"
fi
echo ""

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}📋 Next Steps${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "1. Configure DNS records:"
echo "   Add A record: ${DOMAIN} → ${SERVER_IP}"
echo "   Add A record: *.${DOMAIN} → ${SERVER_IP} (if using DNSPod)"
echo ""
echo "2. Wait for DNS propagation (5-30 minutes)"
echo ""
echo "3. Access admin panel and check generated license:"
echo "   http://${DOMAIN}/admin"
echo ""
echo "4. View logs:"
echo "   $COMPOSE_CMD logs -f hugoverse"
echo ""
echo "5. Stop services:"
echo "   $COMPOSE_CMD down"
echo ""
echo "6. Restart services:"
echo "   $COMPOSE_CMD restart"
echo ""

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}💡 Need More Licenses?${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  Visit: https://mdfriday.com/pricing"
echo "  Purchase a Master License"
echo "  Add to .env: MASTER_LICENSE=YOUR_KEY"
echo "  Restart: $COMPOSE_CMD restart hugoverse"
echo ""

echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}📚 Documentation & Support${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "  GitHub: https://github.com/mdfriday/hugoverse"
echo "  Issues: https://github.com/mdfriday/hugoverse/issues"
echo "  Email: support@mdfriday.com"
echo ""

echo -e "${GREEN}🎉 Happy self-hosting!${NC}"
echo ""
