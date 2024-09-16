// Package cmd provides the kuota-calc command.
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"runtime"
	"text/tabwriter"

	"github.com/druppelt/kuota-calc/internal/calc"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	kuotaCalcExample = `    # provide a simple/complex deployment by piping it to kuota-calc (used as kubectl plugin)
    cat deployment.yaml | kubectl %[1]s

    # do the same, calling the binary directly with detailed output
    cat deployment.yaml | %[1]s --detailed`
)

// KuotaCalcOpts holds all command options.
type KuotaCalcOpts struct {
	genericclioptions.IOStreams

	// flags
	debug       bool
	detailed    bool
	version     bool
	maxRollouts int
	// files    []string

	versionInfo *Version
}

// NewKuotaCalcCmd returns a coba command wrapping KuotaCalcOps
func NewKuotaCalcCmd(version *Version, streams genericclioptions.IOStreams) *cobra.Command {
	opts := KuotaCalcOpts{
		IOStreams:   streams,
		versionInfo: version,
	}

	cmd := &cobra.Command{
		Use:          "kuota-calc",
		Short:        "Calculate the resource quota needs of your deployment(s).",
		Example:      fmt.Sprintf(kuotaCalcExample, "kuota-calc"),
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if opts.version {
				return opts.printVersion()
			}

			return opts.run()
		},
	}

	cmd.Flags().BoolVar(&opts.debug, "debug", false, "enable debug logging")
	cmd.Flags().BoolVar(&opts.detailed, "detailed", false, "enable detailed output")
	cmd.Flags().BoolVar(&opts.version, "version", false, "print version and exit")
	cmd.Flags().IntVar(&opts.maxRollouts, "max-rollouts", -1, "limit the simultaneous rollout to the n most expensive rollouts per resource")

	return cmd
}

func (opts *KuotaCalcOpts) printVersion() error {
	_, _ = fmt.Fprintf(opts.Out, "version %s (revision: %s)\n\tbuild date: %s\n\tgo version: %s\n",
		opts.versionInfo.Version,
		opts.versionInfo.Commit,
		opts.versionInfo.Date,
		runtime.Version(),
	)

	return nil
}

func (opts *KuotaCalcOpts) run() error {
	var (
		summary []*calc.ResourceUsage
	)

	yamlReader := yaml.NewYAMLReader(bufio.NewReader(opts.In))

	for {
		data, err := yamlReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("reading input: %w", err)
		}

		usage, err := calc.ResourceQuotaFromYaml(data)
		if err != nil {
			if errors.Is(err, calc.ErrResourceNotSupported) {
				if opts.debug {
					_, _ = fmt.Fprintf(opts.Out, "DEBUG: %s\n", err)
				}

				continue
			}

			return err
		}

		summary = append(summary, usage)
	}

	if opts.detailed {
		opts.printDetailed(summary)
	} else {
		opts.printSummary(summary)
	}

	return nil
}

func (opts *KuotaCalcOpts) printDetailed(usage []*calc.ResourceUsage) {
	w := tabwriter.NewWriter(opts.Out, 0, 0, 4, ' ', tabwriter.TabIndent)

	_, _ = fmt.Fprintf(w, "Version\tKind\tName\tReplicas\tStrategy\tMaxReplicas\tCPURequest\tCPULimit\tMemoryRequest\tMemoryLimit\t\n")

	for _, u := range usage {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%d\t%s\t%s\t%s\t%s\t\n",
			u.Details.Version,
			u.Details.Kind,
			u.Details.Name,
			u.Details.Replicas,
			u.Details.Strategy,
			u.Details.MaxReplicas,
			u.RolloutResources.CPUMin.String(),
			u.RolloutResources.CPUMax.String(),
			u.RolloutResources.MemoryMin.String(),
			u.RolloutResources.MemoryMax.String(),
		)
	}

	if err := w.Flush(); err != nil {
		_, _ = fmt.Fprintf(opts.Out, "printing detailed resources to tabwriter failed: %v\n", err)
	}

	if opts.maxRollouts > -1 {
		_, _ = fmt.Fprintf(opts.Out, "\nTable assuming simultaneous rollout of all resources\n")
		_, _ = fmt.Fprintf(opts.Out, "Total assuming simultaneous rollout of %d resources\n", opts.maxRollouts)
	} else {
		_, _ = fmt.Fprintf(opts.Out, "\nTable and Total assuming simultaneous rollout of all resources\n")
	}

	_, _ = fmt.Fprintf(opts.Out, "\nTotal\n")

	opts.printSummary(usage)
}

func (opts *KuotaCalcOpts) printSummary(usage []*calc.ResourceUsage) {
	totalResources := calc.Total(opts.maxRollouts, usage)

	_, _ = fmt.Fprintf(opts.Out, "CPU Request: %s\nCPU Limit: %s\nMemory Request: %s\nMemory Limit: %s\n",
		totalResources.CPUMin.String(),
		totalResources.CPUMax.String(),
		totalResources.MemoryMin.String(),
		totalResources.MemoryMax.String(),
	)
}
