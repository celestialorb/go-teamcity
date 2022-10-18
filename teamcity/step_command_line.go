package teamcity

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type ContainerPlatform string

const (
	Any     ContainerPlatform = "*"
	Linux                     = "linux"
	Windows                   = "windows"
)

// ContainerDefinition represents the container configuration that a command
// line step will run within.
type ContainerDefinition struct {
	// Image reference is the container image reference including registry, image,
	// and optionally tag and digest.
	ImageReference string

	// Image platform is the platform the container will run on.
	// e.g. Linux, Windows.
	ImagePlatform ContainerPlatform

	// Whether or not to explicitly pull the image every time the step is run.
	ExplicitlyPullImage bool

	// Additional arguments to add to the container run (i.e. docker run) command.
	AdditionalContainerRunArguments string
}

// StepCommandLine represents a a build step of type "CommandLine"
type StepCommandLine struct {
	ID           string
	Name         string
	stepType     string
	stepJSON     *stepJSON
	isExecutable bool
	//CustomScript contains code for platform specific script, like .cmd on windows or shell script on Unix-like environments.
	CustomScript string
	//CommandExecutable is the executable program to be called from this step.
	CommandExecutable string
	//CommandParameters are additional parameters to be passed on to the CommandExecutable.
	CommandParameters string
	//ExecuteMode is the execute mode for the step. See StepExecuteMode for details.
	ExecuteMode StepExecuteMode
	//Container is the definition of the container the step will run within.
	Container ContainerDefinition
}

// NewStepCommandLineScript creates a command line build step that runs an inline platform-specific script.
func NewStepCommandLineScript(name string, script string) (*StepCommandLine, error) {
	if script == "" {
		return nil, errors.New("script is required")
	}

	return &StepCommandLine{
		Name:         name,
		isExecutable: false,
		stepType:     StepTypeCommandLine,
		CustomScript: script,
		ExecuteMode:  StepExecuteModeDefault,
	}, nil
}

// NewStepCommandLineExecutable creates a command line that invokes an external executable.
func NewStepCommandLineExecutable(name string, executable string, args string) (*StepCommandLine, error) {
	if executable == "" {
		return nil, errors.New("executable is required")
	}

	return &StepCommandLine{
		Name:              name,
		stepType:          StepTypeCommandLine,
		isExecutable:      true,
		CommandExecutable: executable,
		CommandParameters: args,
		ExecuteMode:       StepExecuteModeDefault,
	}, nil
}

func (s *StepCommandLine) GetContainer() ContainerDefinition {
	return s.Container
}

// GetID is a wrapper implementation for ID field, to comply with Step interface
func (s *StepCommandLine) GetID() string {
	return s.ID
}

// GetName is a wrapper implementation for Name field, to comply with Step interface
func (s *StepCommandLine) GetName() string {
	return s.Name
}

// Type returns the step type, in this case "StepTypeCommandLine".
func (s *StepCommandLine) Type() BuildStepType {
	return StepTypeCommandLine
}

func (s *StepCommandLine) properties() *Properties {
	props := NewPropertiesEmpty()
	props.AddOrReplaceValue("teamcity.step.mode", string(s.ExecuteMode))

	if s.isExecutable {
		props.AddOrReplaceValue("command.executable", s.CommandExecutable)

		if s.CommandParameters != "" {
			props.AddOrReplaceValue("command.parameters", s.CommandParameters)
		}
	} else {
		props.AddOrReplaceValue("script.content", s.CustomScript)
		props.AddOrReplaceValue("use.custom.script", "true")
	}

	// TODO: move the container property management to another function.
	// If we don't have a container image reference, don't set any container
	// properties.
	if s.Container.ImageReference != "" {
		// Set the container image property.
		props.AddOrReplaceValue("plugin.docker.imageId", s.Container.ImageReference)

		// Only set the container platform if we've explicitly selected one.
		if s.Container.ImagePlatform != Any {
			props.AddOrReplaceValue("plugin.docker.imagePlatform", string(s.Container.ImagePlatform))
		}

		// Set whether or not to explicitly pull the image.
		props.AddOrReplaceValue("plugin.docker.pull.enabled", strconv.FormatBool(s.Container.ExplicitlyPullImage))

		// If we're given any additional run arguments, go ahead and set them.
		if s.Container.AdditionalContainerRunArguments != "" {
			props.AddOrReplaceValue("plugin.docker.run.parameters", s.Container.AdditionalContainerRunArguments)
		}
	}

	return props
}

func (s *StepCommandLine) serializable() *stepJSON {
	return &stepJSON{
		ID:         s.ID,
		Name:       s.Name,
		Type:       s.stepType,
		Properties: s.properties(),
	}
}

// MarshalJSON implements JSON serialization for StepCommandLine
func (s *StepCommandLine) MarshalJSON() ([]byte, error) {
	out := s.serializable()
	return json.Marshal(out)
}

// UnmarshalJSON implements JSON deserialization for StepCommandLine
func (s *StepCommandLine) UnmarshalJSON(data []byte) error {
	var aux stepJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Type != string(StepTypeCommandLine) {
		return fmt.Errorf("invalid type %s trying to deserialize into StepCommandLine entity", aux.Type)
	}
	s.Name = aux.Name
	s.ID = aux.ID
	s.stepType = StepTypeCommandLine

	props := aux.Properties
	if _, ok := props.GetOk("use.custom.script"); ok {
		s.isExecutable = false
		if v, ok := props.GetOk("script.content"); ok {
			s.CustomScript = v
		}
	}

	if v, ok := props.GetOk("command.executable"); ok {
		s.CommandExecutable = v
		if v, ok := props.GetOk("command.parameters"); ok {
			s.CommandParameters = v
		}
	}

	if v, ok := props.GetOk("teamcity.step.mode"); ok {
		s.ExecuteMode = StepExecuteMode(v)
	}

	if v, ok := props.GetOk("plugin.docker.imageId"); ok {
		s.Container.ImageReference = v
	}

	return nil
}
