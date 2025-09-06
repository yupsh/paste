package paste

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	yup "github.com/yupsh/framework"
	"github.com/yupsh/framework/opt"

	localopt "github.com/yupsh/paste/opt"
)

// Flags represents the configuration options for the paste command
type Flags = localopt.Flags

// Command implementation
type command opt.Inputs[string, Flags]

// Paste creates a new paste command with the given parameters
func Paste(parameters ...any) yup.Command {
	cmd := command(opt.Args[string, Flags](parameters...))
	// Set default delimiter
	if cmd.Flags.Delimiter == "" {
		cmd.Flags.Delimiter = "\t"
	}
	return cmd
}

func (c command) Execute(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer) error {
	// Check for cancellation before starting
	if err := yup.CheckContextCancellation(ctx); err != nil {
		return err
	}

	delimiters := c.getDelimiters()

	if bool(c.Flags.Serial) {
		return c.executeSerial(ctx, stdin, stdout, stderr, delimiters)
	} else {
		return c.executeParallel(ctx, stdin, stdout, stderr, delimiters)
	}
}

func (c command) getDelimiters() []string {
	delimStr := string(c.Flags.Delimiter)
	if delimStr == "" {
		return []string{"\t"}
	}

	var delimiters []string
	for _, char := range delimStr {
		if char == '\\' {
			delimiters = append(delimiters, "\\")
		} else if char == 't' && len(delimiters) > 0 && delimiters[len(delimiters)-1] == "\\" {
			// Handle \t escape sequence
			delimiters[len(delimiters)-1] = "\t"
		} else if char == 'n' && len(delimiters) > 0 && delimiters[len(delimiters)-1] == "\\" {
			// Handle \n escape sequence
			delimiters[len(delimiters)-1] = "\n"
		} else {
			delimiters = append(delimiters, string(char))
		}
	}

	return delimiters
}

func (c command) executeSerial(ctx context.Context, input io.Reader, output, stderr io.Writer, delimiters []string) error {
	// Serial mode: paste each file's lines horizontally
	sources := c.getInputSources(input)

	for i, source := range sources {
		// Check for cancellation before each source
		if err := yup.CheckContextCancellation(ctx); err != nil {
			// Close any remaining files
			for j := i; j < len(sources); j++ {
				if sources[j].file != nil {
					sources[j].file.Close()
				}
			}
			return err
		}

		if i > 0 {
			fmt.Fprintln(output) // Blank line between files
		}

		lines, err := c.readLines(ctx, source.reader)
		if source.file != nil {
			source.file.Close()
		}
		if err != nil {
			fmt.Fprintf(stderr, "paste: %s: %v\n", source.name, err)
			continue
		}

		if len(lines) > 0 {
			result := strings.Join(lines, c.getDelimiter(delimiters, 0))
			fmt.Fprintln(output, result)
		}
	}

	return nil
}

func (c command) executeParallel(ctx context.Context, input io.Reader, output, stderr io.Writer, delimiters []string) error {
	// Parallel mode: paste corresponding lines from all files side by side
	sources := c.getInputSources(input)
	allLines := make([][]string, len(sources))

	// Read all lines from all sources
	for i, source := range sources {
		// Check for cancellation before each source
		if err := yup.CheckContextCancellation(ctx); err != nil {
			// Close any remaining files
			for j := i; j < len(sources); j++ {
				if sources[j].file != nil {
					sources[j].file.Close()
				}
			}
			return err
		}

		lines, err := c.readLines(ctx, source.reader)
		if source.file != nil {
			source.file.Close()
		}
		if err != nil {
			fmt.Fprintf(stderr, "paste: %s: %v\n", source.name, err)
			continue
		}
		allLines[i] = lines
	}

	// Check for cancellation after reading all sources
	if err := yup.CheckContextCancellation(ctx); err != nil {
		return err
	}

	// Find maximum number of lines
	maxLines := 0
	for _, lines := range allLines {
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}

	// Output combined lines
	for lineNum := 0; lineNum < maxLines; lineNum++ {
		// Check for cancellation periodically (every 1000 lines for efficiency)
		if lineNum%1000 == 0 {
			if err := yup.CheckContextCancellation(ctx); err != nil {
				return err
			}
		}

		var fields []string
		for fileNum, lines := range allLines {
			var field string
			if lineNum < len(lines) {
				field = lines[lineNum]
			}
			fields = append(fields, field)

			// Add delimiter between fields (but not after the last field)
			if fileNum < len(allLines)-1 {
				delimiter := c.getDelimiter(delimiters, fileNum)
				fields = append(fields, delimiter)
			}
		}

		result := strings.Join(fields, "")
		if bool(c.Flags.Zero) {
			fmt.Fprint(output, result+"\x00")
		} else {
			fmt.Fprintln(output, result)
		}
	}

	return nil
}

func (c command) getDelimiter(delimiters []string, index int) string {
	if len(delimiters) == 0 {
		return "\t"
	}
	return delimiters[index%len(delimiters)]
}

type inputSource struct {
	reader io.Reader
	file   *os.File
	name   string
}

func (c command) getInputSources(input io.Reader) []inputSource {
	var sources []inputSource

	if len(c.Positional) == 0 {
		sources = append(sources, inputSource{reader: input, name: "stdin"})
	} else {
		for _, filename := range c.Positional {
			if filename == "-" {
				sources = append(sources, inputSource{reader: input, name: "stdin"})
			} else {
				file, err := os.Open(filename)
				if err != nil {
					// We'll handle this error in the calling function
					sources = append(sources, inputSource{reader: nil, name: filename})
				} else {
					sources = append(sources, inputSource{reader: file, file: file, name: filename})
				}
			}
		}
	}

	return sources
}

func (c command) readLines(ctx context.Context, reader io.Reader) ([]string, error) {
	if reader == nil {
		return nil, fmt.Errorf("cannot read from nil reader")
	}

	var lines []string
	scanner := bufio.NewScanner(reader)

	for yup.ScanWithContext(ctx, scanner) {
		line := scanner.Text()
		if bool(c.Flags.Zero) {
			// Handle null-terminated lines
			lines = append(lines, strings.Split(line, "\x00")...)
		} else {
			lines = append(lines, line)
		}
	}

	// Check if context was cancelled
	if err := yup.CheckContextCancellation(ctx); err != nil {
		return lines, err
	}

	return lines, scanner.Err()
}

func (c command) String() string {
	return fmt.Sprintf("paste %v", c.Positional)
}
