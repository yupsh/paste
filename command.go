package command

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	gloo "github.com/gloo-foo/framework"
)

type command gloo.Inputs[string, flags]

func Paste(parameters ...any) gloo.Command {
	cmd := command(gloo.Initialize[string, flags](parameters...))
	// Set default delimiter
	if cmd.Flags.Delimiter == "" {
		cmd.Flags.Delimiter = "\t"
	}
	return cmd
}

func (p command) Executor() gloo.CommandExecutor {
	return func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
		// Get file paths from positional parameters
		filePaths := p.Positional
		if len(filePaths) == 0 {
			// If no files specified, read from stdin
			filePaths = []string{"-"}
		}

		// Open all files
		var scanners []*bufio.Scanner
		var files []*os.File

		for _, path := range filePaths {
			var scanner *bufio.Scanner

			if path == "-" {
				scanner = bufio.NewScanner(stdin)
			} else {
				file, err := os.Open(path)
				if err != nil {
					_, _ = fmt.Fprintf(stderr, "paste: %s: %v\n", path, err)
					return err
				}
				files = append(files, file)
				scanner = bufio.NewScanner(file)
			}

			scanners = append(scanners, scanner)
		}

		// Close files when done
		defer func() {
			for _, f := range files {
				f.Close()
			}
		}()

		delimiter := string(p.Flags.Delimiter)

		// Handle serial mode (-s flag)
		if bool(p.Flags.Serial) {
			// Serial mode: paste all lines from each file sequentially
			for _, scanner := range scanners {
				var lines []string
				for scanner.Scan() {
					lines = append(lines, scanner.Text())
				}
				if err := scanner.Err(); err != nil {
					_, _ = fmt.Fprintf(stderr, "paste: %v\n", err)
					return err
				}

				// Output all lines from this file joined by delimiter
				_, _ = fmt.Fprintln(stdout, strings.Join(lines, delimiter))
			}
		} else {
			// Parallel mode (default): merge corresponding lines from all files
			for {
				var line []string
				anyMore := false

				// Read one line from each file
				for _, scanner := range scanners {
					if scanner.Scan() {
						line = append(line, scanner.Text())
						anyMore = true
					} else {
						// No more lines from this file, use empty string
						line = append(line, "")
					}
				}

				// If no files had any more lines, we're done
				if !anyMore {
					break
				}

				// Output merged line
				_, _ = fmt.Fprintln(stdout, strings.Join(line, delimiter))
			}

			// Check for scanner errors
			for _, scanner := range scanners {
				if err := scanner.Err(); err != nil {
					_, _ = fmt.Fprintf(stderr, "paste: %v\n", err)
					return err
				}
			}
		}

		return nil
	}
}
