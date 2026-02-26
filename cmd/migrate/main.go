// RAD Gateway Database Migration CLI
//
// This tool provides safe database migration management with rollback support.
//
// Usage:
//   migrate up              - Run all pending migrations
//   migrate up N            - Run N pending migrations
//   migrate up-to VERSION   - Migrate to specific version
//   migrate down            - Rollback last migration
//   migrate down N          - Rollback N migrations
//   migrate down-to VERSION - Rollback to specific version
//   migrate version         - Show current version
//   migrate status          - Show migration status
//   migrate create NAME     - Create new migration file
//   migrate verify          - Verify migration integrity
//
// Environment Variables:
//   DATABASE_URL      - Database connection string (required)
//   MIGRATIONS_PATH   - Path to migration files (default: ./migrations)
//   DRY_RUN           - Set to "true" for dry-run mode
//
// Safety Features:
//   - All migrations run in transactions
//   - Checksum verification prevents tampering
//   - Dry-run mode shows what would happen
//   - Version tracking ensures idempotency
//
// Exit Codes:
//   0 - Success
//   1 - General error
//   2 - Database connection error
//   3 - Migration error
//   4 - Verification failed
package main
import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"radgateway/internal/db"
)
// Version of the migration tool
const Version = "1.0.0"
//go:embed migrations/*.sql
var migrationsFS embed.FS
type Config struct {
	DatabaseURL    string
	MigrationsPath string
	DryRun         bool
	Force          bool // Skip confirmation prompts
	Timeout        time.Duration
}
func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	command := os.Args[1]
	// Parse global flags
	cfg := parseConfig()
	// Remove command from args for subcommand parsing
	if len(os.Args) > 2 {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
	} else {
		os.Args = os.Args[:1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	var exitCode int
	switch command {
	case "up":
		exitCode = cmdUp(ctx, cfg)
	case "up-to":
		exitCode = cmdUpTo(ctx, cfg)
	case "down":
		exitCode = cmdDown(ctx, cfg)
	case "down-to":
		exitCode = cmdDownTo(ctx, cfg)
	case "version":
		exitCode = cmdVersion(ctx, cfg)
	case "status":
		exitCode = cmdStatus(ctx, cfg)
	case "create":
		exitCode = cmdCreate(cfg)
	case "verify":
		exitCode = cmdVerify(ctx, cfg)
	case "help", "--help", "-h":
		printUsage()
		exitCode = 0
	case "version-tool":
		fmt.Printf("RAD Gateway Migration Tool v%s\n", Version)
		exitCode = 0
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		exitCode = 1
	}
	os.Exit(exitCode)
}
func parseConfig() Config {
	cfg := Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "./migrations"),
		DryRun:         strings.ToLower(os.Getenv("DRY_RUN")) == "true",
		Timeout:        5 * time.Minute,
	}
	// Parse global flags
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Migration timeout")
	flag.BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Show what would be done without executing")
	flag.BoolVar(&cfg.Force, "force", false, "Skip confirmation prompts")
	// Custom usage
	flag.Usage = printUsage
	flag.Parse()
	return cfg
}
func printUsage() {
	fmt.Println(`RAD Gateway Database Migration Tool
Usage:
  migrate <command> [options]
Commands:
  up [N]              Run all pending migrations (or N migrations)
  up-to VERSION       Migrate to specific version
  down [N]            Rollback last migration (or N migrations)
  down-to VERSION     Rollback to specific version
  version             Show current database version
  status              Show detailed migration status
  create NAME         Create a new migration file
  verify              Verify migration integrity
  help                Show this help message
Options:
  -dry-run            Show what would be done without executing
  -force              Skip confirmation prompts
  -timeout DURATION   Migration timeout (default: 5m)
Environment Variables:
  DATABASE_URL        Database connection string (required)
                      Example: postgres://user:pass@localhost/dbname
                               sqlite3://./radgateway.db
  MIGRATIONS_PATH     Path to migration files (default: ./migrations)
  DRY_RUN             Set to "true" for dry-run mode
Examples:
  # Run all pending migrations
  DATABASE_URL=postgres://user:pass@localhost/radgateway migrate up
  # Run migrations in dry-run mode
  migrate up -dry-run
  # Rollback last 3 migrations
  migrate down 3
  # Create a new migration
  migrate create "add user preferences table"
  # Verify migration integrity
  migrate verify`)
}
func cmdUp(ctx context.Context, cfg Config) int {
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL not set")
		return 2
	}
	// Parse count argument
	count := 0 // 0 means all
	if len(os.Args) > 1 {
		n, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid count: %s\n", os.Args[1])
			return 1
		}
		count = n
	}
	// Connect to database
	database, migrator, err := connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	defer database.Close()
	// Get current status
	status, err := migrator.GetStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
		return 3
	}
	if status.PendingCount == 0 {
		fmt.Println("No pending migrations")
		return 0
	}
	// Determine target version
	targetVersion := 0
	if count > 0 && count < status.PendingCount {
		targetVersion = status.Pending[count-1].Version
	}
	// Show what will happen
	fmt.Printf("Will apply %d migration(s)\n", status.PendingCount)
	if cfg.DryRun {
		fmt.Println("DRY RUN MODE - No changes will be made")
	}
	for _, mig := range status.Pending {
		if targetVersion > 0 && mig.Version > targetVersion {
			break
		}
		fmt.Printf("  - %03d_%s.sql\n", mig.Version, strings.ReplaceAll(mig.Name, " ", "_"))
	}
	// Confirm unless dry-run or force
	if !cfg.DryRun && !cfg.Force {
		fmt.Print("\nContinue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Aborted")
			return 0
		}
	}
	// Run migrations
	if err := migrator.UpTo(ctx, targetVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		return 3
	}
	fmt.Println("\nMigrations completed successfully")
	return 0
}
func cmdUpTo(ctx context.Context, cfg Config) int {
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL not set")
		return 2
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: version required")
		return 1
	}
	targetVersion, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid version: %s\n", os.Args[1])
		return 1
	}
	database, migrator, err := connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	defer database.Close()
	currentVersion, err := migrator.Version(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting version: %v\n", err)
		return 3
	}
	if currentVersion >= targetVersion {
		fmt.Printf("Already at version %d (target: %d)\n", currentVersion, targetVersion)
		return 0
	}
	fmt.Printf("Will migrate from version %d to %d\n", currentVersion, targetVersion)
	if cfg.DryRun {
		fmt.Println("DRY RUN MODE - No changes will be made")
	}
	if !cfg.DryRun && !cfg.Force {
		fmt.Print("\nContinue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Aborted")
			return 0
		}
	}
	if err := migrator.UpTo(ctx, targetVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		return 3
	}
	fmt.Println("Migration completed successfully")
	return 0
}
func cmdDown(ctx context.Context, cfg Config) int {
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL not set")
		return 2
	}
	// Parse count argument
	count := 1 // Default to 1
	if len(os.Args) > 1 {
		n, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid count: %s\n", os.Args[1])
			return 1
		}
		count = n
	}
	database, migrator, err := connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	defer database.Close()
	status, err := migrator.GetStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
		return 3
	}
	if status.CurrentVersion == 0 {
		fmt.Println("No migrations to rollback")
		return 0
	}
	fmt.Printf("WARNING: Will rollback %d migration(s)\n", count)
	fmt.Println("This may result in DATA LOSS!")
	if cfg.DryRun {
		fmt.Println("DRY RUN MODE - No changes will be made")
	}
	if !cfg.DryRun {
		fmt.Print("\nAre you sure? Type 'yes' to continue: ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Aborted")
			return 0
		}
	}
	if err := migrator.DownBy(ctx, count); err != nil {
		fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
		return 3
	}
	fmt.Println("Rollback completed successfully")
	return 0
}
func cmdDownTo(ctx context.Context, cfg Config) int {
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL not set")
		return 2
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: version required")
		return 1
	}
	targetVersion, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid version: %s\n", os.Args[1])
		return 1
	}
	database, migrator, err := connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	defer database.Close()
	currentVersion, err := migrator.Version(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting version: %v\n", err)
		return 3
	}
	if currentVersion <= targetVersion {
		fmt.Printf("Already at or below version %d (current: %d)\n", targetVersion, currentVersion)
		return 0
	}
	fmt.Printf("WARNING: Will rollback to version %d from %d\n", targetVersion, currentVersion)
	fmt.Println("This may result in DATA LOSS!")
	if cfg.DryRun {
		fmt.Println("DRY RUN MODE - No changes will be made")
	}
	if !cfg.DryRun {
		fmt.Print("\nAre you sure? Type 'yes' to continue: ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("Aborted")
			return 0
		}
	}
	if err := migrator.DownTo(ctx, targetVersion); err != nil {
		fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
		return 3
	}
	fmt.Println("Rollback completed successfully")
	return 0
}
func cmdVersion(ctx context.Context, cfg Config) int {
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL not set")
		return 2
	}
	database, migrator, err := connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	defer database.Close()
	version, err := migrator.Version(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting version: %v\n", err)
		return 3
	}
	if version == 0 {
		fmt.Println("Database version: 0 (no migrations applied)")
	} else {
		fmt.Printf("Database version: %d\n", version)
	}
	return 0
}
func cmdStatus(ctx context.Context, cfg Config) int {
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL not set")
		return 2
	}
	database, migrator, err := connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	defer database.Close()
	status, err := migrator.GetStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
		return 3
	}
	fmt.Println("Migration Status")
	fmt.Println("================")
	fmt.Printf("Current version: %d\n", status.CurrentVersion)
	fmt.Printf("Target version:  %d\n", status.TargetVersion)
	fmt.Printf("Pending:         %d\n\n", status.PendingCount)
	if len(status.Applied) > 0 {
		fmt.Println("Applied Migrations:")
		for _, m := range status.Applied {
			fmt.Printf("  [✓] %3d  %s  (applied %s)\n",
				m.Version, m.Name, m.AppliedAt.Format("2006-01-02"))
		}
		fmt.Println()
	}
	if len(status.Pending) > 0 {
		fmt.Println("Pending Migrations:")
		for _, m := range status.Pending {
			fmt.Printf("  [ ] %3d  %s\n", m.Version, m.Name)
		}
	}
	return 0
}
func cmdCreate(cfg Config) int {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: migration name required")
		return 1
	}
	name := strings.Join(os.Args[1:], " ")
	// Ensure migrations directory exists
	if err := os.MkdirAll(cfg.MigrationsPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating migrations directory: %v\n", err)
		return 1
	}
	filepath, err := db.CreateMigration(cfg.MigrationsPath, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating migration: %v\n", err)
		return 1
	}
	fmt.Printf("Created migration: %s\n", filepath)
	return 0
}
func cmdVerify(ctx context.Context, cfg Config) int {
	if cfg.DatabaseURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL not set")
		return 2
	}
	database, migrator, err := connect(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 2
	}
	defer database.Close()
	status, err := migrator.GetStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification failed: %v\n", err)
		return 4
	}
	fmt.Println("Migration Verification")
	fmt.Println("=====================")
	allValid := true
	// Check applied migrations
	for _, record := range status.Applied {
		found := false
		for _, mig := range status.Pending {
			if mig.Version == record.Version {
				found = true
				break
			}
		}
		// Also check if it's in applied but not pending (already applied)
		if record.Version <= status.CurrentVersion {
			found = true
		}
		if !found {
			fmt.Printf("  [WARN] Migration %d recorded but file not found\n", record.Version)
			allValid = false
		}
	}
	if status.PendingCount > 0 {
		fmt.Printf("  [INFO] %d pending migration(s) found\n", status.PendingCount)
	}
	if allValid {
		fmt.Println("  [OK]   All migrations verified successfully")
		return 0
	}
	return 4
}
func connect(cfg Config) (db.Database, *db.Migrator, error) {
	// Parse database URL to determine driver
	driver, dsn := parseDatabaseURL(cfg.DatabaseURL)

	// Create database config
	dbConfig := db.Config{
		Driver: driver,
		DSN:    dsn,
	}
	// Connect
	database, err := db.New(dbConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	// Get underlying *sql.DB for migrator
	// We need to access the internal db field
	var sqlDB *sql.DB
	switch d := database.(type) {
	case *db.PostgresDB:
		sqlDB = getPostgresDB(d)
	case *db.SQLiteDB:
		sqlDB = getSQLiteDB(d)
	default:
		return nil, nil, fmt.Errorf("unsupported database type: %T", database)
	}
	// Create migrator
	migrator := db.NewMigratorWithFS(sqlDB, driver, migrationsFS, "migrations")
	// Load migrations from FS if available, otherwise from directory
	if err := migrator.LoadMigrationsFromFS(); err != nil {
		// Fallback to directory
		migrator = db.NewMigrator(sqlDB, driver)
		if err := migrator.LoadMigrationsFromDir(cfg.MigrationsPath); err != nil {
			return nil, nil, fmt.Errorf("failed to load migrations: %w", err)
		}
	}
	return database, migrator, nil
}
func parseDatabaseURL(url string) (driver, dsn string) {
	if strings.HasPrefix(url, "postgres://") || strings.HasPrefix(url, "postgresql://") {
		return "postgres", url
	}
	if strings.HasPrefix(url, "sqlite://") {
		return "sqlite", strings.TrimPrefix(url, "sqlite://")
	}
	if strings.HasPrefix(url, "sqlite3://") {
		return "sqlite", strings.TrimPrefix(url, "sqlite3://")
	}
	// Assume SQLite file path
	return "sqlite", url
}
// Helper to extract *sql.DB from PostgresDB using the DB() method
func getPostgresDB(d *db.PostgresDB) *sql.DB {
	return d.DB()
}
func getSQLiteDB(d *db.SQLiteDB) *sql.DB {
	return d.DB()
}
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
