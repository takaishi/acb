package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/takaishi/acb/internal/aws"
	"github.com/takaishi/acb/internal/config"
)

func List(cli *CLI) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Validate AWS credentials
	if err := config.ValidateAWSCredentials(ctx); err != nil {
		return err
	}

	// Initialize Cognito client
	cognitoClient, err := aws.NewCognitoClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Cognito client: %w", err)
	}

	// List user pools
	pools, err := cognitoClient.ListUserPools(ctx, cli.List.Pattern)
	if err != nil {
		return fmt.Errorf("failed to list user pools: %w", err)
	}

	// Display results
	if len(pools) == 0 {
		fmt.Println("No user pools found matching the specified pattern")
		return nil
	}

	fmt.Printf("User Pool List (Total: %d)\n", len(pools))
	fmt.Println("----------------------------------------")
	for _, pool := range pools {
		fmt.Printf("ID: %s\nName: %s\n\n", *pool.Id, *pool.Name)
	}

	return nil
}
