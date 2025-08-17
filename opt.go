package command

type Delimiter string

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

type flags struct {
	Delimiter Delimiter
	Serial    SerialFlag
	Zero      ZeroFlag
}

func (d Delimiter) Configure(flags *flags)  { flags.Delimiter = d }
func (s SerialFlag) Configure(flags *flags) { flags.Serial = s }
func (z ZeroFlag) Configure(flags *flags)   { flags.Zero = z }
