package opt

// Custom types for parameters
type Delimiter string

// Boolean flag types with constants
type SerialFlag bool
const (
	Serial   SerialFlag = true
	NoSerial SerialFlag = false
)

type ZeroFlag bool
const (
	Zero   ZeroFlag = true
	NoZero ZeroFlag = false
)

// Flags represents the configuration options for the paste command
type Flags struct {
	Delimiter Delimiter  // List of delimiter characters (-d)
	Serial    SerialFlag // Paste one file at a time instead of in parallel (-s)
	Zero      ZeroFlag   // Line delimiter is NUL, not newline (-z)
}

// Configure methods for the opt system
func (d Delimiter) Configure(flags *Flags)  { flags.Delimiter = d }
func (s SerialFlag) Configure(flags *Flags) { flags.Serial = s }
func (z ZeroFlag) Configure(flags *Flags)   { flags.Zero = z }
