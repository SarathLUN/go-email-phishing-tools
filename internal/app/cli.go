package app

import (
	"context"
	"fmt"
	"github.com/SarathLUN/go-email-phishing-tools/internal/config"
	"github.com/SarathLUN/go-email-phishing-tools/internal/csvutil" // Adjust module path
	"github.com/SarathLUN/go-email-phishing-tools/internal/domain"  // Adjust module path
	"github.com/SarathLUN/go-email-phishing-tools/internal/store"   // Adjust module path
	"github.com/SarathLUN/go-email-phishing-tools/internal/store/sqlite"
	"github.com/joho/godotenv"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	// Add other global flags if needed
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "email-phishing-tools",
	Short: "A CLI tool for simulating email phishing attacks",
	Long: `email-phishing-tools allows you to import targets, send simulation emails,
and track clicks via a simple web service.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// You can add initialization steps here that apply to all commands
		// e.g., loading config if not already done by specific commands
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags here, e.g., for config file path
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.env)")

	// Add subcommands
	addImportCommand()
	// Add other commands (send, serve) here later
}

// --- Import Command Implementation ---

func addImportCommand() {
	var importCmd = &cobra.Command{
		Use:   "import <csv_file_path>",
		Short: "Import targets from a CSV file",
		Long: `Imports target users from a specified CSV file into the database.
The CSV file must contain 'full_name' and 'email' columns.
Existing emails in the database will be skipped.`,
		Args: cobra.ExactArgs(1), // Requires exactly one argument: the CSV file path
		RunE: func(cmd *cobra.Command, args []string) error {
			csvFilePath := args[0]

			// Load configuration
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// --- *** MODIFIED INITIALIZATION *** ---
			// Directly use the sqlite package for connection and repository creation
			db, err := sqlite.ConnectDB(cfg.DBPath) // Use sqlite.ConnectDB
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			// Create the repository using the concrete type constructor from sqlite package
			// but store it in a variable typed as the store.TargetRepository interface
			var targetRepo store.TargetRepository             // Use the interface type
			targetRepo = sqlite.NewSQLiteTargetRepository(db) // Assign the concrete implementation

			// --- Command Logic (remains the same) ---
			log.Printf("Starting import from CSV file: %s", csvFilePath)

			parsedTargets, err := csvutil.ParseTargetsCSV(csvFilePath)
			if err != nil {
				return fmt.Errorf("failed to parse CSV file: %w", err)
			}

			if len(parsedTargets) == 0 {
				log.Println("No valid targets found in CSV to import.")
				return nil
			}

			targetsToCreate := make([]*domain.Target, 0, len(parsedTargets))
			for _, pt := range parsedTargets {
				targetsToCreate = append(targetsToCreate, domain.NewTarget(pt.FullName, pt.Email))
			}

			// Use the targetRepo interface variable here
			insertedCount, err := targetRepo.BulkCreate(context.Background(), targetsToCreate)
			if err != nil {
				return fmt.Errorf("error during bulk insert: %w", err)
			}

			log.Printf("Successfully imported %d new targets into the database.", insertedCount)
			log.Printf("Total records processed from CSV: %d", len(parsedTargets))

			return nil
		},
	}
	rootCmd.AddCommand(importCmd)
}

// --- Helper for goose integration (optional but clean) ---
// We needed this earlier for goose CLI setup

func GetDBPathFromConfig(configPath string) string {
	// Simplified load just for the DB path - avoids full init
	if configPath != "" {
		_ = godotenv.Load(configPath)
	} else {
		_ = godotenv.Load()
	}
	return getEnv("DB_PATH", "./phishing_simulation.db") // Use same helper as config
}

// Helper function to get env var or default (copied from config for standalone use)
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// Add print-db-path command for goose helper
func init() {
	var printDbPathCmd = &cobra.Command{
		Use:    "print-db-path",
		Short:  "Prints the database path based on config (for goose)",
		Args:   cobra.NoArgs,
		Hidden: true, // Hide this utility command from standard help
		Run: func(cmd *cobra.Command, args []string) {
			// Use the global --config flag if provided
			fmt.Print(GetDBPathFromConfig(cfgFile))
		},
	}
	rootCmd.AddCommand(printDbPathCmd)
}
