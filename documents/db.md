# Database Integration for Production

## Overview

This document outlines the database strategy to transform Cloud-Assist from a prototype to a production-ready application with persistent data, user management, and analytics.

## Current Limitations

- **Sessions**: Lost on agent restart 
- **Conversations**: No history persistence
- **Users**: No profile or preference storage
- **Analytics**: No usage tracking or audit trails

## Production Benefits with Database

- ✅ **Persistent Sessions**: Resume conversations across restarts
- ✅ **User Profiles**: Personalized preferences and settings  
- ✅ **Conversation History**: Full chat history and context retention
- ✅ **Audit Trails**: Complete action logging for compliance
- ✅ **Analytics**: Usage patterns and performance insights
- ✅ **Scalability**: Support multiple concurrent users

## Database Architecture

### 1. Primary Database: PostgreSQL
**Rationale**: ACID compliance, JSON support, excellent Python/Go integration

#### Core Tables Structure

```sql
-- Users and Authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    preferences JSONB DEFAULT '{}'
);

-- Sessions Management
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_token VARCHAR(512) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN DEFAULT true
);

-- Conversations
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    title VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    is_archived BOOLEAN DEFAULT false
);

-- Messages within conversations
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}', -- tool calls, attachments, etc.
    parent_message_id UUID REFERENCES messages(id),
    tokens_used INTEGER DEFAULT 0
);

-- Agent State Persistence
CREATE TABLE agent_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    state_type VARCHAR(100) NOT NULL, -- 'graph_state', 'tool_state', etc.
    state_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Audit Trail
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    details JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Performance Analytics
CREATE TABLE usage_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    session_id UUID REFERENCES sessions(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

#### Indexes for Performance

```sql
-- Session management indexes
CREATE INDEX idx_sessions_token ON sessions(session_token);
CREATE INDEX idx_sessions_user_active ON sessions(user_id, is_active);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

-- Conversation indexes
CREATE INDEX idx_conversations_user ON conversations(user_id, created_at DESC);
CREATE INDEX idx_conversations_session ON conversations(session_id);

-- Message indexes
CREATE INDEX idx_messages_conversation ON messages(conversation_id, timestamp);
CREATE INDEX idx_messages_parent ON messages(parent_message_id);

-- Audit trail indexes
CREATE INDEX idx_audit_user_time ON audit_logs(user_id, timestamp DESC);
CREATE INDEX idx_audit_action ON audit_logs(action, timestamp DESC);

-- Analytics indexes
CREATE INDEX idx_analytics_user_time ON usage_analytics(user_id, timestamp DESC);
CREATE INDEX idx_analytics_event ON usage_analytics(event_type, timestamp DESC);
```

### 2. Cache Layer: Redis
**Purpose**: Session caching, rate limiting, real-time data

#### Redis Data Structures

```redis
# Session Cache
session:{session_token} -> {user_id, expires_at, preferences}
TTL: session duration

# User Activity
user_activity:{user_id} -> {last_seen, active_conversations}
TTL: 1 hour

# Rate Limiting
rate_limit:{user_id}:{endpoint} -> request_count
TTL: rate limit window

# Real-time Agent State
agent_state:{conversation_id} -> {current_state, tools_in_use}
TTL: conversation timeout

# Conversation Context Cache
conv_context:{conversation_id} -> {recent_messages, context_summary}
TTL: 24 hours
```

## Implementation Strategy

### Phase 1: Core Infrastructure
1. **Database Setup**
   - PostgreSQL installation and configuration
   - Redis setup for caching
   - Database migration system
   - Connection pooling

2. **Data Access Layer**
   - Repository pattern implementation
   - Database models/entities
   - Connection management
   - Transaction handling

### Phase 2: Session Management
1. **User Authentication**
   - Database-backed user storage
   - Session persistence
   - Token management

2. **Conversation Persistence**
   - Message storage
   - Context retention
   - State management

### Phase 3: Analytics & Monitoring
1. **Audit Trail**
   - Action logging
   - Performance tracking
   - Usage analytics

2. **Monitoring**
   - Database health metrics
   - Query performance
   - Resource utilization

## Technology Stack

### Python Agent (LangGraph)
```python
# Dependencies to add to pyproject.toml
dependencies = [
    "asyncpg>=0.29.0",      # PostgreSQL async driver
    "redis>=5.0.0",         # Redis client
    "sqlalchemy>=2.0.0",    # ORM
    "alembic>=1.13.0",      # Database migrations
    "pydantic>=2.5.0",      # Data validation
]
```

### Go CLI
```go
// Dependencies to add to go.mod
require (
    github.com/lib/pq v1.10.9           // PostgreSQL driver
    github.com/go-redis/redis/v8 v8.11.5 // Redis client
    github.com/golang-migrate/migrate/v4 // Database migrations
    github.com/jmoiron/sqlx v1.3.5      // SQL extensions
)
```

## Database Operations

### Connection Configuration

#### Python Configuration
```python
# database/config.py
from pydantic_settings import BaseSettings

class DatabaseConfig(BaseSettings):
    postgres_url: str = "postgresql://user:pass@localhost:5432/cloudassist"
    redis_url: str = "redis://localhost:6379/0"
    pool_size: int = 10
    max_overflow: int = 20
    pool_timeout: int = 30
    
    class Config:
        env_file = ".env"
```

#### Go Configuration
```go
// internal/config/database.go
type DatabaseConfig struct {
    PostgresURL string `env:"POSTGRES_URL" envDefault:"postgresql://user:pass@localhost:5432/cloudassist"`
    RedisURL    string `env:"REDIS_URL" envDefault:"redis://localhost:6379/0"`
    MaxConns    int    `env:"DB_MAX_CONNS" envDefault:"10"`
    MaxIdle     int    `env:"DB_MAX_IDLE" envDefault:"5"`
}
```

### Data Models

#### Python Models (SQLAlchemy)
```python
# models/user.py
from sqlalchemy import Column, String, Boolean, DateTime, UUID
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.ext.declarative import declarative_base

Base = declarative_base()

class User(Base):
    __tablename__ = "users"
    
    id = Column(UUID(as_uuid=True), primary_key=True, server_default=text("gen_random_uuid()"))
    username = Column(String(255), unique=True, nullable=False)
    email = Column(String(255), unique=True, nullable=False)
    password_hash = Column(String(255), nullable=False)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    preferences = Column(JSONB, default={})
    is_active = Column(Boolean, default=True)
```

#### Go Models
```go
// internal/models/user.go
type User struct {
    ID           string                 `db:"id" json:"id"`
    Username     string                 `db:"username" json:"username"`
    Email        string                 `db:"email" json:"email"`
    PasswordHash string                 `db:"password_hash" json:"-"`
    CreatedAt    time.Time             `db:"created_at" json:"created_at"`
    Preferences  map[string]interface{} `db:"preferences" json:"preferences"`
    IsActive     bool                  `db:"is_active" json:"is_active"`
}
```

## Migration Strategy

### Database Migrations
```sql
-- migrations/001_initial_schema.up.sql
-- Contains all the CREATE TABLE statements above

-- migrations/001_initial_schema.down.sql
DROP TABLE IF EXISTS usage_analytics;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS agent_states;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
```

### Data Migration Plan
1. **Backup Current State**: Export any existing session data
2. **Schema Creation**: Run initial migrations
3. **Data Import**: Import backed up data with proper transformations
4. **Validation**: Verify data integrity
5. **Cutover**: Switch to database-backed operations

## Production Considerations

### Security
- **Encryption**: All sensitive data encrypted at rest
- **Access Control**: Role-based database access
- **Audit Trail**: Complete action logging
- **Data Retention**: Configurable data lifecycle policies

### Performance
- **Connection Pooling**: Efficient database connections
- **Query Optimization**: Proper indexing and query patterns
- **Caching Strategy**: Redis for frequently accessed data
- **Monitoring**: Database performance metrics

### Scalability
- **Read Replicas**: For read-heavy workloads
- **Partitioning**: Time-based partitioning for large tables
- **Archival**: Automated old data archival
- **Backup Strategy**: Regular automated backups

### Monitoring & Alerts
```yaml
# monitoring/database_alerts.yml
alerts:
  - name: "High Database Connections"
    condition: "db_connections > 80% of max"
    severity: "warning"
  
  - name: "Slow Queries"
    condition: "query_duration > 5s"
    severity: "critical"
  
  - name: "Failed Connections"
    condition: "connection_failures > 10/minute"
    severity: "critical"
```

## Implementation Timeline

### Week 1: Infrastructure Setup
- [ ] PostgreSQL and Redis installation
- [ ] Database schema creation
- [ ] Migration system setup
- [ ] Basic connection testing

### Week 2: Data Access Layer
- [ ] Repository pattern implementation
- [ ] Model definitions (Python & Go)
- [ ] Basic CRUD operations
- [ ] Connection pooling

### Week 3: Session Management
- [ ] User authentication with database
- [ ] Session persistence
- [ ] Conversation storage
- [ ] State management

### Week 4: Integration & Testing
- [ ] Agent integration
- [ ] CLI integration
- [ ] Performance testing
- [ ] Security validation

### Week 5: Production Readiness
- [ ] Monitoring setup
- [ ] Backup procedures
- [ ] Documentation completion
- [ ] Deployment scripts

## Configuration Files

### Environment Variables
```bash
# .env.production
POSTGRES_URL=postgresql://cloudassist:secure_password@db:5432/cloudassist_prod
REDIS_URL=redis://redis:6379/0
DB_POOL_SIZE=20
DB_MAX_OVERFLOW=30
REDIS_POOL_SIZE=10
SESSION_TIMEOUT=3600
AUDIT_RETENTION_DAYS=90
```

### Docker Compose for Database Services
```yaml
# docker-compose.database.yml
version: '3.8'
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: cloudassist
      POSTGRES_USER: cloudassist
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./database/init:/docker-entrypoint-initdb.d
    
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

volumes:
  postgres_data:
  redis_data:
```

## Next Steps

1. **Review and Approve**: Review this database strategy
2. **Environment Setup**: Prepare development database environment
3. **Implementation Start**: Begin with Phase 1 infrastructure
4. **Testing Strategy**: Define comprehensive testing approach
5. **Production Planning**: Plan production deployment strategy

This database integration will transform Cloud-Assist from a stateless prototype to a production-ready application with persistent data, user management, and comprehensive analytics capabilities.
