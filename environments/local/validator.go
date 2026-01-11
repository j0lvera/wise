package local

import (
	"fmt"
	"regexp"

	"github.com/j0lvera/wise/environments"
)

// DefaultBlockedPatterns contains patterns for dangerous commands.
var DefaultBlockedPatterns = []string{
	`rm\s+-[rf]*\s+/`,           // rm -rf / or rm -r / or rm -f /
	`rm\s+-[rf]*\s+\*`,          // rm -rf * or similar
	`rm\s+-[rf]*\s+~`,           // rm -rf ~
	`>\s*/dev/sd`,               // writing to disk devices
	`mkfs`,                      // formatting filesystems
	`dd\s+if=.*/dev/`,           // dd from devices
	`dd\s+of=.*/dev/`,           // dd to devices
	`chmod\s+777\s+/`,           // chmod 777 on root
	`chown\s+-R\s+.*\s+/`,       // recursive chown on root
	`curl.*\|\s*(ba)?sh`,        // curl | sh (pipe to shell)
	`wget.*\|\s*(ba)?sh`,        // wget | sh
	`:\(\)\{\s*:\|:&\s*\};:`,    // fork bomb
	`/dev/null\s*>\s*/etc/`,     // overwriting /etc files
	`>\s*/etc/passwd`,           // overwriting passwd
	`>\s*/etc/shadow`,           // overwriting shadow
	`shutdown`,                  // system shutdown
	`reboot`,                    // system reboot
	`init\s+0`,                  // system halt
	`halt`,                      // system halt
	`poweroff`,                  // power off
}

// BlocklistValidator blocks commands matching dangerous patterns.
type BlocklistValidator struct {
	patterns []*regexp.Regexp
}

// NewBlocklistValidator creates a validator with the given patterns.
func NewBlocklistValidator(patterns []string) (*BlocklistValidator, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return &BlocklistValidator{patterns: compiled}, nil
}

// NewDefaultValidator creates a validator with default dangerous patterns.
func NewDefaultValidator() environments.CommandValidator {
	v, _ := NewBlocklistValidator(DefaultBlockedPatterns)
	return v
}

// Validate checks if the command matches any blocked pattern.
func (v *BlocklistValidator) Validate(command string) error {
	for _, re := range v.patterns {
		if re.MatchString(command) {
			return &ExecutionError{
				Type:    ErrBlocked,
				Message: fmt.Sprintf("Command blocked for safety: matches pattern %q. Please use a safer alternative.", re.String()),
			}
		}
	}
	return nil
}
