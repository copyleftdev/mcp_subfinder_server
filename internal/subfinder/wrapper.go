// Package subfinder provides a wrapper for the subfinder library
package subfinder

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/projectdiscovery/goflags"
	"github.com/projectdiscovery/subfinder/v2/pkg/runner"
)

type SubfinderConfig struct {
	ProviderConfigPath    string
	Timeout               int
	MaxDepth              int
	SourcesFilter         string
	ExcludeSourcesFilter  string
	Recursive             bool
}

func RunEnumeration(ctx context.Context, domain string, config SubfinderConfig, logger *slog.Logger) ([]string, error) {
	if config.Timeout <= 0 {
		config.Timeout = 120
	}

	runnerOpts := &runner.Options{
		Silent:             false,
		RemoveWildcard:     true,
		Timeout:            config.Timeout,
		MaxEnumerationTime: config.Timeout,
		Threads:            40,
		All:                true,
		CaptureSources:     true,
		ProviderConfig:     config.ProviderConfigPath,
		Resolvers:          nil,
		Verbose:            true,
	}

	if config.SourcesFilter != "" {
		sources := goflags.StringSlice{}
		for _, source := range strings.Split(config.SourcesFilter, ",") {
			sources.Set(strings.TrimSpace(source))
		}
		runnerOpts.Sources = sources
		runnerOpts.All = false
	}

	if config.ExcludeSourcesFilter != "" {
		excludeSources := goflags.StringSlice{}
		for _, source := range strings.Split(config.ExcludeSourcesFilter, ",") {
			excludeSources.Set(strings.TrimSpace(source))
		}
		runnerOpts.ExcludeSources = excludeSources
	}

	logger.Info("Initializing subfinder with options", 
		"timeout", config.Timeout,
		"recursive", config.Recursive,
		"allSources", runnerOpts.All)

	subfinderRunner, err := runner.NewRunner(runnerOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create subfinder runner: %w", err)
	}

	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(config.Timeout)*time.Second)
		defer cancel()
	}

	outputBuffer := &bytes.Buffer{}

	maxRetries := 3
	var resultMap map[string]map[string]struct{}
	var enumErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Info("Starting subdomain enumeration", 
			"domain", domain, 
			"attempt", attempt, 
			"recursive", config.Recursive)
		
		startTime := time.Now()
		
		resultMap, enumErr = subfinderRunner.EnumerateSingleDomainWithCtx(ctx, domain, []io.Writer{outputBuffer})
		
		elapsedTime := time.Since(startTime)
		logger.Info("Enumeration attempt completed", 
			"domain", domain, 
			"attempt", attempt,
			"durationMs", elapsedTime.Milliseconds(), 
			"resultsCount", len(resultMap))
		
		if enumErr == nil {
			bufferContent := outputBuffer.String()
			if len(bufferContent) > 0 {
				logger.Debug("Subfinder output", "output", bufferContent)
			}
		}
		
		if enumErr == nil && len(resultMap) > 0 {
			break
		}
		
		select {
		case <-ctx.Done():
			logger.Warn("Context cancelled, stopping retries")
			if enumErr == nil {
				enumErr = ctx.Err()
			}
			break
		default:
			if attempt < maxRetries {
				logger.Warn("Retry attempt failed, trying again", 
					"attempt", attempt, 
					"error", enumErr,
					"resultsCount", len(resultMap))
				time.Sleep(2 * time.Second)
			}
		}
	}

	if enumErr != nil {
		return nil, fmt.Errorf("enumeration error after %d attempts: %w", maxRetries, enumErr)
	}

	var subdomains []string
	for subdomain := range resultMap {
		if strings.EqualFold(subdomain, domain) {
			continue
		}
		subdomains = append(subdomains, subdomain)
	}

	sort.Strings(subdomains)

	for _, subdomain := range subdomains {
		sources := resultMap[subdomain]
		var sourceNames []string
		for source := range sources {
			sourceNames = append(sourceNames, source)
		}
		sort.Strings(sourceNames)
		logger.Debug("Subdomain sources", 
			"subdomain", subdomain, 
			"sources", strings.Join(sourceNames, ","))
	}

	if config.Recursive && len(subdomains) > 0 && config.MaxDepth > 1 {
		logger.Info("Starting recursive enumeration", "foundSubdomains", len(subdomains))
		
		maxSubdomainsToProcess := 10
		if len(subdomains) > maxSubdomainsToProcess {
			logger.Info("Limiting recursive processing", 
				"total", len(subdomains), 
				"processing", maxSubdomainsToProcess)
			subdomainsToProcess := subdomains[:maxSubdomainsToProcess]
			subdomains = append(subdomainsToProcess, subdomains[maxSubdomainsToProcess:]...)
		}
		
		allSubdomains := make(map[string]struct{})
		for _, subdomain := range subdomains {
			allSubdomains[subdomain] = struct{}{}
		}
		
		maxDepth := config.MaxDepth
		if maxDepth <= 0 {
			maxDepth = 2
		}
		
		recursiveTimeout := config.Timeout / 2
		if recursiveTimeout < 30 {
			recursiveTimeout = 30
		}
		
		recursiveOpts := *runnerOpts
		recursiveOpts.Timeout = recursiveTimeout
		recursiveOpts.MaxEnumerationTime = recursiveTimeout
		
		recursiveRunner, err := runner.NewRunner(&recursiveOpts)
		if err != nil {
			logger.Warn("Failed to create recursive runner", "error", err)
		} else {
			for i, subdomain := range subdomains {
				if i >= maxSubdomainsToProcess {
					break
				}
				
				logger.Info("Recursively checking", "subdomain", subdomain)
				
				recursiveCtx, cancel := context.WithTimeout(context.Background(), 
					time.Duration(recursiveTimeout)*time.Second)
				
				recursiveBuffer := &bytes.Buffer{}
				
				recResultMap, recErr := recursiveRunner.EnumerateSingleDomainWithCtx(
					recursiveCtx, subdomain, []io.Writer{recursiveBuffer})
				
				cancel()
				
				if recErr != nil {
					logger.Warn("Error in recursive enumeration", 
						"subdomain", subdomain, "error", recErr)
					continue
				}
				
				for recSubdomain := range recResultMap {
					if strings.EqualFold(recSubdomain, subdomain) {
						continue
					}
					if _, exists := allSubdomains[recSubdomain]; exists {
						continue
					}
					allSubdomains[recSubdomain] = struct{}{}
					logger.Info("Found recursive subdomain", "subdomain", recSubdomain)
				}
			}
		}
		
		subdomains = make([]string, 0, len(allSubdomains))
		for subdomain := range allSubdomains {
			subdomains = append(subdomains, subdomain)
		}
		sort.Strings(subdomains)
	}

	if len(subdomains) == 0 {
		logger.Warn("No subdomains found via passive enumeration")
		
		commonPrefixes := []string{"www", "mail", "api", "dev", "blog", "shop", "app", "support", "help", "portal"}
		logger.Info("Suggesting common subdomains to check", 
			"prefixes", strings.Join(commonPrefixes, ", "))
		
		for _, prefix := range commonPrefixes {
			commonSubdomain := prefix + "." + domain
			subdomains = append(subdomains, commonSubdomain)
		}
	}

	logger.Info("Enumeration complete", 
		"domain", domain, 
		"subdomainsFound", len(subdomains))
	
	stats := subfinderRunner.GetStatistics()
	if stats != nil {
		logger.Info("Enumeration statistics", 
			"totalSources", len(stats))
		
		var successfulSources []string
		for source, stat := range stats {
			if stat.Results > 0 {
				successfulSources = append(successfulSources, 
					fmt.Sprintf("%s:%d", source, stat.Results))
			}
		}
		
		if len(successfulSources) > 0 {
			logger.Info("Successful sources", 
				"sources", strings.Join(successfulSources, ", "))
		}
	}

	return subdomains, nil
}
