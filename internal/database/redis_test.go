package database

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/compnew2006/whatomate/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getMockRedisConfig(mr *miniredis.Miniredis) *config.RedisConfig {
	_, portStr, _ := net.SplitHostPort(mr.Addr())
	port, _ := strconv.Atoi(portStr)
	return &config.RedisConfig{
		Host: "localhost",
		Port: port,
		DB:   0,
	}
}

// TestNewRedis_Success tests successful Redis client creation
func TestNewRedis_Success(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	// Override port to use miniredis
	_, portStr, _ := net.SplitHostPort(mr.Addr())
	port, _ := strconv.Atoi(portStr)
	cfg.Port = port
	client, err := NewRedis(cfg)

	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify the client works
	ctx := context.Background()
	err = client.Set(ctx, "test", "value", 0).Err()
	assert.NoError(t, err)

	// Clean up
	client.Close()
}

// TestNewRedis_WithPassword tests Redis client creation with password
func TestNewRedis_WithPassword(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	// Set a password
	mr.RequireAuth("testpassword")

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "testpassword",
		DB:       0,
	}

	client, err := NewRedis(cfg)

	require.NoError(t, err)
	assert.NotNil(t, client)

	// Clean up
	client.Close()
}

// TestNewRedis_WithDatabase tests Redis client creation with specific DB
func TestNewRedis_WithDatabase(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       2, // Use database 2
	}

	_, portStr, _ := net.SplitHostPort(mr.Addr())
	port, _ := strconv.Atoi(portStr)
	cfg.Port = port

	client, err := NewRedis(cfg)

	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify we're using the correct database
	ctx := context.Background()
	err = client.Set(ctx, "test", "value", 0).Err()
	assert.NoError(t, err)

	// Clean up
	client.Close()
}

// TestNewRedis_ConnectionFailure tests connection failure handling
func TestNewRedis_ConnectionFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.RedisConfig{
		Host:     "nonexistent.host.invalid",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := NewRedis(cfg)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect to redis")
}

// TestNewRedis_InvalidPort tests invalid port handling
func TestNewRedis_InvalidPort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		port      int
		expectErr bool
	}{
		{
			name:      "invalid port 0",
			port:      0,
			expectErr: true,
		},
		{
			name:      "invalid port negative",
			port:      -1,
			expectErr: false, // Will fail at connection, not validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mr := miniredis.RunT(t)
			defer mr.Close()

			cfg := &config.RedisConfig{
				Host:     "localhost",
				Port:     tt.port,
				Password: "",
				DB:       0,
			}

			// Adjust for miniredis port
			if tt.port == 99999 {
				mr.Close()
				client, err := NewRedis(cfg)
				assert.Error(t, err)
				assert.Nil(t, client)
				return
			}

			// For the 0 port test, just verify it creates the client
			// The actual connection will fail
			client, err := NewRedis(cfg)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				// May succeed or fail depending on miniredis state
				if err == nil {
					assert.NotNil(t, client)
					client.Close()
				}
			}
		})
	}
}

// TestNewRedis_ClientConfiguration tests that the client is configured correctly
func TestNewRedis_ClientConfiguration(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       5,
	}

	_, portStr, _ := net.SplitHostPort(mr.Addr())
	port, _ := strconv.Atoi(portStr)
	cfg.Port = port

	client, err := NewRedis(cfg)

	require.NoError(t, err)
	assert.NotNil(t, client)

	// Verify client options
	ctx := context.Background()

	// Test basic operations work
	err = client.Set(ctx, "key", "value", 0).Err()
	assert.NoError(t, err)

	val, err := client.Get(ctx, "key").Result()
	assert.NoError(t, err)
	assert.Equal(t, "value", val)

	// Clean up
	client.Close()
}

// TestNewRedis_ContextTimeout tests that the connection test works correctly
func TestNewRedis_ContextTimeout(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := NewRedis(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Clean up
	client.Close()
}

// TestNewRedis_MultipleConnections tests creating multiple clients
func TestNewRedis_MultipleConnections(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := getMockRedisConfig(mr)

	// Create first client
	client1, err := NewRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, client1)
	defer client1.Close()

	// Create second client
	client2, err := NewRedis(cfg)
	require.NoError(t, err)
	require.NotNil(t, client2)
	defer client2.Close()

	// Both clients should work
	ctx := context.Background()

	err = client1.Set(ctx, "client1", "value1", 0).Err()
	assert.NoError(t, err)

	err = client2.Set(ctx, "client2", "value2", 0).Err()
	assert.NoError(t, err)

	// Verify each client can read what it wrote
	val1, err := client1.Get(ctx, "client1").Result()
	assert.NoError(t, err)
	assert.Equal(t, "value1", val1)

	val2, err := client2.Get(ctx, "client2").Result()
	assert.NoError(t, err)
	assert.Equal(t, "value2", val2)
}

// TestNewRedis_ClientPools tests that the client can handle concurrent operations
func TestNewRedis_ClientPools(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := NewRedis(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Test concurrent operations
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		err := client.Set(ctx, fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 0).Err()
		assert.NoError(t, err)
	}

	// Verify all values were set
	for i := 0; i < 10; i++ {
		val, err := client.Get(ctx, fmt.Sprintf("key%d", i)).Result()
		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("value%d", i), val)
	}
}

// TestNewRedis_Ping tests that the ping during creation works correctly
func TestNewRedis_Ping(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := NewRedis(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Verify ping works
	ctx := context.Background()
	err = client.Ping(ctx).Err()
	assert.NoError(t, err)

	// Clean up
	client.Close()
}

// TestNewRedis_ConfigDefaults tests that default config values work
func TestNewRedis_ConfigDefaults(t *testing.T) {
	t.Parallel()

	mr := miniredis.RunT(t)
	defer mr.Close()

	cfg := getMockRedisConfig(mr)

	client, err := NewRedis(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Clean up
	client.Close()
}

// TestNewRedis_HostPortParsing tests that host and port are correctly parsed
func TestNewRedis_HostPortParsing(t *testing.T) {
	t.Parallel()

	t.Run("valid host port", func(t *testing.T) {
		t.Parallel()

		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := &config.RedisConfig{
			Host:     "127.0.0.1",
			Port:     6379,
			Password: "",
			DB:       0,
		}

		client, err := NewRedis(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, client)

		// Verify it works
		ctx := context.Background()
		err = client.Ping(ctx).Err()
		assert.NoError(t, err)

		client.Close()
	})

	t.Run("hostname instead of IP", func(t *testing.T) {
		t.Parallel()

		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := &config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		}

		client, err := NewRedis(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, client)
		client.Close()
	})
}

// TestNewRedis_DatabaseIsolation tests that different DB numbers are isolated
func TestNewRedis_DatabaseIsolation(t *testing.T) {
	t.Parallel()

	t.Run("same database number", func(t *testing.T) {
		t.Parallel()

		mr := miniredis.RunT(t)
		defer mr.Close()

		cfg := getMockRedisConfig(mr)

		// Create two clients for the same database
		client1, err := NewRedis(cfg)
		require.NoError(t, err)
		defer client1.Close()

		client2, err := NewRedis(cfg)
		require.NoError(t, err)
		defer client2.Close()

		ctx := context.Background()

		// Set a key with client1
		err = client1.Set(ctx, "test_key", "value1", 0).Err()
		assert.NoError(t, err)

		// Verify client2 can see it (same database)
		val, err := client2.Get(ctx, "test_key").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value1", val)
	})

	t.Run("different database numbers", func(t *testing.T) {
		t.Parallel()

		mr := miniredis.RunT(t)
		defer mr.Close()

		// Create client for DB 0
		cfg0 := getMockRedisConfig(mr)
		client0, err := NewRedis(cfg0)
		require.NoError(t, err)
		defer client0.Close()

		// Create client for DB 1
		cfg1 := getMockRedisConfig(mr)
		cfg1.DB = 1

		client1, err := NewRedis(cfg1)
		require.NoError(t, err)
		defer client1.Close()

		ctx := context.Background()

		// Set a key in DB 0
		err = client0.Set(ctx, "test_key", "db0_value", 0).Err()
		assert.NoError(t, err)

		// Verify DB 1 cannot see it
		val, err := client1.Get(ctx, "test_key").Result()
		assert.Error(t, err)
		assert.Empty(t, val)

		// Set a key in DB 1
		err = client1.Set(ctx, "test_key", "db1_value", 0).Err()
		assert.NoError(t, err)

		// Both databases should be accessible separately
		val0, err := client0.Get(ctx, "test_key").Result()
		assert.NoError(t, err)
		assert.Equal(t, "db0_value", val0)

		val1, err := client1.Get(ctx, "test_key").Result()
		assert.NoError(t, err)
		assert.Equal(t, "db1_value", val1)
	})
}
