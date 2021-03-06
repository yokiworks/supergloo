package routerule

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/solo-io/supergloo/cli/pkg/common"

	"github.com/solo-io/solo-kit/pkg/errors"

	"github.com/gogo/protobuf/types"
	"github.com/solo-io/supergloo/cli/pkg/cmd/options"
	"github.com/solo-io/supergloo/cli/pkg/common/iutil"
)

func EnsureDuration(rootMessage string, durOpts *options.InputDuration, targetDur *types.Duration, opts *options.Options) error {
	dur := types.Duration{}
	ensureDuration := func(ans interface{}) error {
		switch val := ans.(type) {
		case string:
			_, err := time.ParseDuration(val)
			if err != nil {
				return err
			}
		default:
			return errors.Errorf("Incorrect format for duration, must be a string")
		}
		return nil
	}
	if !opts.Top.Static && opts.Top.File == "" {
		err := iutil.GetStringInput(rootMessage+common.DurationQuestionExample, durOpts, ensureDuration)
		if err != nil {
			return err
		}
	}
	if *durOpts != "" {
		duration, err := time.ParseDuration(*durOpts)
		if err != nil {
			return err
		}
		dur.Seconds = int64(duration.Seconds())
		dur.Nanos = int32(duration.Nanoseconds() % time.Second.Nanoseconds())
	}
	*targetDur = dur
	return nil
}

// EnsurePercentage transforms a source string to a target int
// If not present, it promts the user for input with the given message
// Errors on invalid input
func EnsurePercentage(message string, source *string, target *int32, opts *options.Options) error {
	ensurePercentage := func(ans interface{}) error {
		switch val := ans.(type) {
		case string:
			v, err := strconv.Atoi(val)
			if err != nil {
				return err
			}
			if v < 0 || v > 100 {
				return errors.Errorf("percent values must be between 0-100")
			}
		default:
			return errors.Errorf("val (%s) is the incorrect format")
		}
		return nil
	}
	if !opts.Top.Static && opts.Top.File == "" {
		if err := iutil.GetStringInput(message, source, ensurePercentage); err != nil {
			return err
		}
	}
	if *source != "" {
		percentage, err := strconv.Atoi(*source)
		if err != nil {
			return err
		}
		if percentage < 0 || percentage > 100 {
			return errors.Errorf("percent values must be between 0-100")
		}
		*target = int32(percentage)
	}
	return nil
}

func ensureCsv(message string, source string, target *[]string, staticMode bool, required bool) error {
	if staticMode && required && source == "" {
		return fmt.Errorf(message)
	}
	if !staticMode {
		if err := iutil.GetStringInput(message, &source, nil); err != nil {
			return err
		}
	}
	parts := strings.Split(source, ",")
	*target = parts
	return nil
}

// Expected format of source: k1,v1,k2,v2
func ensureKVCsv(message string, source string, target *map[string]string, staticMode bool, required bool) error {
	parts := []string{}
	ensureCsv(message, source, &parts, staticMode, required)
	if len(parts)%2 != 0 {
		return fmt.Errorf("Must provide one key per value (received an odd sum)")
	}
	for i := 0; i < len(parts)/2; i++ {
		(*target)[parts[i*2]] = parts[i*2+1]
	}
	return nil
}
