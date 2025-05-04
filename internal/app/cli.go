package app

import (
	"context"
	"fmt"
	"github.com/SarathLUN/go-email-phishing-tools/internal/config"
	"github.com/SarathLUN/go-email-phishing-tools/internal/csvutil" // Adjust module path
	"github.com/SarathLUN/go-email-phishing-tools/internal/domain"  // Adjust module path
	"github.com/SarathLUN/go-email-phishing-tools/internal/email"
	"github.com/SarathLUN/go-email-phishing-tools/internal/store" // Adjust module path
	"github.com/SarathLUN/go-email-phishing-tools/internal/store/sqlite"
	"github.com/joho/godotenv"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

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
	addSendCommand()        // *** ADD THIS CALL ***
	addPrintDbPathCommand() // Add other commands (serve) here later
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
func addPrintDbPathCommand() {
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

// --- Send Command Implementation ---

func addSendCommand() {
	var sendCmd = &cobra.Command{
		Use:   "send",
		Short: "Send phishing simulation emails to non-sent targets",
		Long: `Finds all targets in the database that have not yet received the simulation
email (sent_at is NULL) and sends them a personalized email using the configured
template and SMTP server. Updates the sent_at timestamp upon success.`,
		Args: cobra.NoArgs, // No arguments needed for this command
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load configuration
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load configuration: %w", err)
			}

			// --- Validate required Send config ---
			if cfg.SMTPUser == "" || cfg.SMTPPassword == "" || cfg.SMTPSenderAddress == "" {
				return fmt.Errorf("SMTP configuration (SMTP_USER, SMTP_PASSWORD, SMTP_SENDER_ADDRESS) is incomplete in config. Cannot send emails")
			}
			if cfg.EmailTemplatePath == "" {
				return fmt.Errorf("email template path (EMAIL_TEMPLATE_PATH) is not configured")
			}
			if _, err := os.Stat(cfg.EmailTemplatePath); os.IsNotExist(err) {
				return fmt.Errorf("email template file not found at path: %s", cfg.EmailTemplatePath)
			}
			if cfg.TrackerBaseURL == "" {
				return fmt.Errorf("tracker base URL (TRACKER_BASE_URL) is not configured")
			}

			// Initialize dependencies (DB, Repo, Email Sender)
			db, err := sqlite.ConnectDB(cfg.DBPath)
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			var targetRepo store.TargetRepository
			targetRepo = sqlite.NewSQLiteTargetRepository(db)

			emailSender, err := email.NewGmailSender(cfg) // Initialize sender
			if err != nil {
				return fmt.Errorf("failed to initialize email sender: %w", err)
			}

			// --- Command Logic ---
			log.Println("Starting email sending process...")
			ctx := context.Background()

			// 1. Find non-sent targets
			targets, err := targetRepo.FindNonSent(ctx)
			if err != nil {
				return fmt.Errorf("failed to retrieve non-sent targets: %w", err)
			}

			if len(targets) == 0 {
				log.Println("No targets found awaiting emails. Nothing to do.")
				return nil
			}

			log.Printf("Found %d targets to send emails to.", len(targets))

			// 2. Iterate and send
			successCount := 0
			failCount := 0
			for _, target := range targets {
				log.Printf("Processing target: %s (%s)", target.FullName, target.Email)

				// Construct unique tracking link
				trackingLink, err := buildTrackingLink(cfg.TrackerBaseURL, target.UUID.String())
				if err != nil {
					log.Printf("ERROR: Failed to build tracking link for %s (%s): %v. Skipping.", target.FullName, target.Email, err)
					failCount++
					continue // Skip this target
				}

				// Prepare template data
				templateData := email.EmailTemplateData{
					FullName:     target.FullName,
					TrackingLink: trackingLink,
					// Subject could also be dynamic if needed
				}

				// Send email
				err = emailSender.Send(target.Email, target.FullName, cfg.EmailSubject, templateData)
				if err != nil {
					log.Printf("ERROR: Failed to send email to %s (%s): %v", target.FullName, target.Email, err)
					failCount++
					continue // Skip marking as sent if email failed
				}

				// Mark as sent in DB
				sentTime := time.Now()
				err = targetRepo.MarkAsSent(ctx, target.UUID, sentTime)
				if err != nil {
					// CRITICAL: Email sent but DB update failed. Log prominently.
					log.Printf("CRITICAL ERROR: Email sent to %s (%s) but failed to mark as sent in DB (UUID: %s): %v", target.FullName, target.Email, target.UUID, err)
					// Technically counted as success because email went out, but state is inconsistent.
					// Consider how to handle this - maybe retry DB update later? For now, log and count success.
					// Let's count as failure for reporting consistency, as the process didn't fully complete.
					failCount++
					// successCount++ // Or count success but log critical error
				} else {
					log.Printf("Successfully processed and marked target %s (%s) as sent.", target.FullName, target.Email)
					successCount++
				}

				// Add delay
				time.Sleep(1 * time.Second) // Send one email per second (adjust as needed)
			}

			log.Println("--------------------------------------------------")
			log.Printf("Email Sending Summary:")
			log.Printf("  Targets processed: %d", len(targets))
			log.Printf("  Successfully sent: %d", successCount)
			log.Printf("  Failed/Skipped:    %d", failCount)
			log.Println("--------------------------------------------------")

			return nil
		},
	}
	rootCmd.AddCommand(sendCmd)
}

// Helper function to build the tracking link safely
func buildTrackingLink(baseURL, uuid string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid TRACKER_BASE_URL '%s': %w", baseURL, err)
	}

	// Ensure the path ends with a slash if not empty, for proper joining
	if base.Path != "" && !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}
	// Define the tracking endpoint path
	trackingPath := "feedback" // Or make this configurable?

	// Add query parameter
	query := base.Query()
	query.Set("id", uuid) // Use 'id' as the parameter name

	// Reconstruct URL - JoinPath is safer for paths
	finalURL, err := url.JoinPath(baseURL, trackingPath)
	if err != nil {
		return "", fmt.Errorf("failed to join path '%s' to base URL '%s': %w", trackingPath, baseURL, err)
	}

	finalURL += "?" + query.Encode() // Append query string

	return finalURL, nil
}
