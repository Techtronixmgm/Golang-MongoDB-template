## Go / Gin / MongoDB Backend Boilerplate

### Core

 - Go + Gin

 - MongoDB

 - JWT authentication

 - Configuration management

 - REST API

 - Environment-based configuration

### Authentication & Authorization

 - Login / Logout

 - JWT access & refresh tokens

 - Token refresh / session management

 - Password hashing

 - Role-based access control

 - Roles: `Admin`, `Customer`

### Customer

 - Register

    -  Username

    -  Email

    -  Alternative email (optional)

    -  Password

 - Login with username/email

 - Logout

 - View/update profile

 - Change password

 - Refresh session

### Admin

 - View/update own profile

 - Change own password

 - Create customer

 - Update customer

 - Search customers

 - Get customer by ID

 - Change customer status

 - Change customer password

### Database

 - MongoDB connection

 - Connection management

 - Indexes

 - Repository layer

### Security

 - Password hashing

 - Access/refresh token rotation

 - Input validation

 - Rate limiting

 - CORS

 - Authentication & authorization middleware

### Boilerplate Essentials

 - Standard API response/error format

 - Request validation

 - Pagination & filtering

 - Logging

 - Health check

 - Graceful shutdown

`.env.example`

### .env

```
MONGO_URI=mongodb://localhost:27017/
MONGO_DB_NAME=<db_name>
PORT=<serverport>
JWT_SECRET=<secret_long_string>
JWT_EXPIRY_HOURS=24
GIN_MODE=debug
# or release
#ALLOWED_ORIGINS=http://localhost:<frontend-serverport>
AUTH_ACCESS_COOKIE=access_token
AUTH_REFRESH_COOKIE=refresh_token

REFRESH_TOKEN_EXPIRY_DAYS=30
COOKIE_SECURE=false
COOKIE_SAME_SITE=lax


```


```
Initial System Setup

1. Register the first user.
2. Update the user's role in MongoDB to "admin".
3. Log in again as admin.
4. Make neccesasry changes in the code

```

