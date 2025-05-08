# Environment Variables (.env)

This document explains how the environment variables in the `.env` file are used throughout the microservices project.

## Usage

The `.env` file contains environment variables used by the Docker Compose setup. These variables are referenced in the `docker-compose.yml` file using the `${VARIABLE_NAME}` syntax.

## Setup Instructions

1. Make a copy of the `.env` file if you need different configurations:
   ```bash
   cp .env .env.local
   ```

2. For production, create a specific environment file:
   ```bash
   cp .env .env.production
   ```

3. To use a specific environment file with Docker Compose:
   ```bash
   docker-compose --env-file .env.production up -d
   ```

## Environment Variables

### Server/Application Environment
- `SERVER_ENV`: The environment the server is running in (development, production, etc.)
- `APP_ENV`: Alternative environment variable used by some services
- `SERVER_PORT`: The port each service runs on inside its container

### Database Configuration
- `DB_HOST`: Database host address
- `DB_PORT`: Database port
- `DB_USER`: Database user
- `DB_PASSWORD`: Database password
- `DB_NAME`: Database name
- `DB_SSL_MODE`: SSL mode for database connection

### JWT Configuration
- `JWT_SECRET`: Secret key for JWT token generation
- `JWT_EXPIRATION_HOURS`: JWT expiration time in hours

### OAuth Configuration
- `TOKEN_SECRET`: Secret for OAuth token generation
- `ACCESS_TOKEN_EXPIRATION_MINUTES`: Access token expiration in minutes
- `REFRESH_TOKEN_EXPIRATION_DAYS`: Refresh token expiration in days

### Service URLs
- `OAUTH_BASE_URL`: Base URL for the OAuth service
- `SUPPLIER_SERVICE_URL`: URL for the Supplier Service

### Client Credentials
- `MERCHANT_CLIENT_ID`: Client ID for Merchant Service
- `MERCHANT_CLIENT_SECRET`: Client Secret for Merchant Service
- `PRODUCT_CLIENT_ID`: Client ID for Product Service
- `PRODUCT_CLIENT_SECRET`: Client Secret for Product Service
- `SUPPLIER_CLIENT_ID`: Client ID for Supplier Service
- `SUPPLIER_CLIENT_SECRET`: Client Secret for Supplier Service

### Grafana Configuration
- `GF_SECURITY_ADMIN_PASSWORD`: Admin password for Grafana
- `GF_USERS_ALLOW_SIGN_UP`: Setting to allow user signup in Grafana

## Security Best Practices

1. **Never commit the `.env` file to version control**. Add it to your `.gitignore` file.
2. For production environments, use a secrets management solution instead of plain text environment variables.
3. Rotate secrets regularly, especially in production environments.
4. Use different secrets for different environments.

## Example .gitignore entry:

```
# Environment variables
.env
.env.*
!.env.example
```