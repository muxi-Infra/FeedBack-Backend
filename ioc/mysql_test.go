package ioc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	mysqlclient "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestMySQLStartupErrorPreservesSafeDiagnostics(t *testing.T) {
	const sensitive = "fictional-sensitive-driver-message"
	for _, tt := range []struct {
		name string
		err  error
		want string
	}{
		{"authentication", &mysqlclient.MySQLError{Number: 1045, Message: sensitive}, "mysql_code=1045"},
		{"migration permission", &mysqlclient.MySQLError{Number: 1142, Message: sensitive}, "mysql_code=1142"},
		{"deadline", context.DeadlineExceeded, "error_class=timeout"},
		{"cancellation", context.Canceled, "error_class=canceled"},
		{"dns", &net.DNSError{Err: sensitive, Name: sensitive}, "error_class=dns"},
		{"network", &net.OpError{Op: "dial", Net: "tcp", Err: errors.New(sensitive)}, "error_class=network"},
		{"network timeout", &net.DNSError{Err: sensitive, Name: sensitive, IsTimeout: true}, "error_class=timeout"},
		{"unknown driver error", errors.New(sensitive), "error_class=driver"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Wrapping must preserve classification without exposing either message.
			err := mysqlStartupError("initialization", fmt.Errorf("%s: %w", sensitive, tt.err))
			require.EqualError(t, err, "mysql initialization failed ("+tt.want+")")
			require.NotContains(t, err.Error(), sensitive)
		})
	}
	require.EqualError(t, mysqlStartupError("migration", &mysqlclient.MySQLError{Number: 1142, Message: sensitive}), "mysql migration failed (mysql_code=1142)")
}
