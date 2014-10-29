package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"code.google.com/p/go-uuid/uuid"

	"github.com/jessevdk/go-flags"
)

// Options are the  CLI options used by go-flags
type Options struct {
	// Package to build
	Package string `short:"p" long:"package" description:"package to build relative to cwd" env:"PACKAGE" required:"true"`
	// Template file to use
	Template string `short:"t" long:"template" description:"Input template to parse will Dockerfile.in in the container dir" env:"TEMPLATE"`
	// Version to build
	Version string `short:"v" long:"version" description:"Major Version" env:"VERSION" required:"true"`
	// Revison to add to build
	Rev string `short:"r" long:"rev" description:"Revsion" env:"REV" default:"0"`
	// Location of docker binary
	Docker string `short:"d" long:"docker-bin" description:"Path to the docker binary. We look in $PATH by default" env:"DOCKER_BIN"`
	// OS's to build
	BuildOS []string `short:"b" long:"build-os" description:"specify the os's too buld can use multiple times. Use comma sepparated list if using ENV Variable" env:"BUILDOS" env-delim:"," default:"el5" default:"el6" default:"el7"`
	// Disable layer caching
	DisableCache bool `short:"c" long:"disable-cache" description:"disable Docker layer caching for builds" default:"false" env:"DISABLE_CACHE"`
	// populated by the build loop for the curent os. since we pass options into the template
	OS string
}

var opts Options

// FindBin returns the path to the current runnign binary
func FindBin() (dir string) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return
}

// findDocker looks for docker and bails if it can't find it
func findDocker() (path string) {
	path, err := exec.LookPath("docker")
	if err != nil {
		log.Fatal("Couldn't find docker in $PATH use -d to specify the path to docker binary.")
	}
	path, _ = filepath.Abs(path)
	return
}

// cliDefaults sets the options to default vaules if env or switches haven't specified them
// TODO: use go-flags callbacks for these
func cliDefaults() {
	if opts.Template == "" {
		opts.Template = fmt.Sprintf("%s/%s/Dockerfile.in", FindBin(), opts.Package)
	}

	if opts.Version == "" {
		fmt.Println("Please Specify a version to build")
		os.Exit(1)
	}

	if opts.Docker == "" {
		opts.Docker = findDocker()
	}
}

func fatalExec(cmd *exec.Cmd) {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	fmt.Println("EXEC:", cmd.Args)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Issue running %s \n%s", cmd.Args, err.Error())
	}
}

func main() {
	// process flags
	if _, err := flags.Parse(&opts); err != nil {
		if err.(*flags.Error).Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}
	cliDefaults()

	// open template file for processing
	inputTemplate, err := ioutil.ReadFile(opts.Template)
	if err != nil {
		fmt.Println("Template File: ", opts.Template)
		log.Fatalf("Error reading templatefile: %s \n %s", opts.Template, err.Error())
	}

	// assemble template
	t := template.Must(template.New("letter").Parse(string(inputTemplate)))

	binPath := FindBin()
	// Set CWD to project dir
	pkgDir := binPath + "/" + opts.Package
	err = os.Chdir(pkgDir)
	if err != nil {
		fmt.Printf("Error chaning directory to %s\n%s", pkgDir, err.Error())
	}

	// loop over platforms and build
	for _, platform := range opts.BuildOS {
		opts.OS = platform
		buff := new(bytes.Buffer)
		err := t.Execute(buff, opts)
		if err != nil {
			log.Fatalf("Error compiling template: %s\n %s", opts.Template, err.Error())
		}

		// TODO: might be able to call this from docker golib direct
		// This builds the docker container from the compiled template \m/
		// store and use the generated dockerfile
		container := opts.Package + ":build_" + opts.OS
		cmd := exec.Command(opts.Docker, "build", "-t", container, ".")
		err = ioutil.WriteFile(fmt.Sprintf("%s/Dockerfile", pkgDir), buff.Bytes(), 0640)
		if err != nil {
			log.Fatalf("Error writing generated dockerfile for %s.\n%s", pkgDir, err.Error())
		}

		if opts.DisableCache == true {
			cmd = exec.Command(opts.Docker, "build", "-t", container, "--no-cache=true", ".")
		}
		// build it!
		fatalExec(cmd)

		// start the container so we can get the rpm
		instance := opts.Package + uuid.New()
		fatalExec(exec.Command(opts.Docker, "run", "--name", instance, container))

		// pull the package
		// example instance:/collectd-5.4.1/collectd-5.4.1-1.mine-el6.x86_64.rpm
		pkgFile := instance + ":/" + opts.Package + "/" + opts.Package + "-" + opts.Version + "-" + opts.Rev + ".x86_64.rpm"
		localFile := binPath + "/pkg/" + opts.Package + "/" + opts.OS + "/"

		fatalExec(exec.Command(opts.Docker, "cp", pkgFile, localFile))

		// stop the container
		fatalExec(exec.Command(opts.Docker, "stop", instance))
	}
}
