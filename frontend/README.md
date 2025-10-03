# LogBuilder Frontend

A React TypeScript application for viewing and querying application logs with authentication.

## Features

- **User Authentication**: Login and signup with JWT tokens
- **Protected Routes**: Routes are protected and require authentication
- **Log Viewer**: Query logs using natural language questions
- **Real-time Search**: Search through logs with the intelligent query API
- **Responsive UI**: Built with Tailwind CSS for a modern, responsive design

## Getting Started

### Prerequisites

- Node.js 14+ and npm
- LogBuilder backend API running on `http://localhost:8080`

### Installation

1. Install dependencies:
```bash
npm install
```

2. Create a `.env` file (see `.env.example`):
```bash
REACT_APP_API_URL=http://localhost:8080/api/v1
```

### Running the App

Start the development server:
```bash
npm start
```

The app will open at [http://localhost:3000](http://localhost:3000)

### Building for Production

```bash
npm run build
```

## Project Structure

```
src/
├── components/         # Reusable components
│   ├── Navbar.tsx     # Navigation bar
│   └── ProtectedRoute.tsx  # Route protection wrapper
├── contexts/          # React contexts
│   └── AuthContext.tsx     # Authentication state management
├── pages/             # Page components
│   ├── Login.tsx      # Login page
│   ├── Register.tsx   # Registration page
│   └── Logs.tsx       # Log viewer page
├── services/          # API services
│   ├── api.ts         # Axios instance with interceptors
│   ├── auth.ts        # Authentication service
│   └── logs.ts        # Logs service
└── App.tsx            # Main app component with routing
```

## Usage

### Register a New Account

1. Navigate to the register page
2. Enter username (min 3 characters), email, and password (min 8 characters)
3. Click "Sign up"

### Login

1. Navigate to the login page
2. Enter your username and password
3. Click "Sign in"

### Query Logs

Once logged in, you can query your logs using natural language:

- "Show me all errors from today"
- "Find warnings in the last hour"
- "Show info logs from service X"
- "Get debug logs between 2pm and 3pm"

The results will display:
- Log level (color-coded)
- Timestamp
- Source
- Message
- Metadata (expandable)

## Authentication

The app uses JWT tokens for authentication:
- Tokens are stored in `localStorage`
- The API service automatically attaches tokens to requests
- 401 responses automatically redirect to login
- Users can logout to clear their session

## API Integration

The frontend connects to the LogBuilder backend API:
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login user
- `POST /api/v1/logs/query` - Query logs with natural language

## Technologies

- **React 18** - UI framework
- **TypeScript** - Type safety
- **React Router** - Navigation
- **Axios** - HTTP client
- **Tailwind CSS** - Styling
- **JWT** - Authentication
