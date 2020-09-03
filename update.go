//IxOS-update is used to trigger online updates of an ixia
//  chassis. This requires the chassis to have an Internet Connection
//  includes sftp support for automatic update package upload

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"time"

	"github.com/pkg/sftp"
	"github.com/schollz/progressbar"
	"golang.org/x/crypto/ssh"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"gioui.org/font/gofont"
)

var host = "10.0.0.1"
var username string
var password string
var offlineUpdatePackage string
var version string

//buffer to support function feedback in the GUI
var output = bytes.NewBufferString("")

//runCommand takes a commands and provides the response to the console
//  assumes the client is already connected
func runCommand(conn *ssh.Client, command string) bytes.Buffer {

	//maybe validate that conn is live?

	var stdoutBuf bytes.Buffer

	//create the session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	session.Stdout = &stdoutBuf
	session.Run(command)

	//pause for effect
	time.Sleep(4 * time.Second)

	return stdoutBuf
}

//runCommand takes an array of commands and provides the response to the console
//  assumes the client is already connected
func runCommands(conn *ssh.Client, commands ...string) bytes.Buffer {

	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	in, err := session.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	var out bytes.Buffer
	session.Stdout = &out
	session.Stderr = &out // this will send stderr to the same buffer

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		log.Fatal(err)
	}
	err = session.Shell()
	if err != nil {
		log.Fatal(err)
	}
	// send the commands
	var stdoutBuf string
	for _, command := range commands {
		_, err = fmt.Fprintf(in, "%s\n", command)
		if err != nil {
			fmt.Println("Failed to run command: ", err)
			log.Fatal(err)
		}
		time.Sleep(4 * time.Second)

		stdoutBuf = out.String()

		//check for prompt
		for bytes.HasSuffix([]byte(stdoutBuf), []byte("# ")) == false {
			time.Sleep(60 * time.Second)
			stdoutBuf += out.String()
			fmt.Printf("Waiting for command to complete: %s", stdoutBuf[len(stdoutBuf)-50:])
		}
	}

	//return our output
	return out
}

func uploadFile(client *sftp.Client, filename string) error {

	// create source file
	srcFile, err := os.Open("./" + filename)
	if err != nil {
		return err
	}

	srcFileInfo, err := os.Stat("./" + filename)
	if err != nil {
		return err
	}

	// create destination file
	dstFile, err := client.Create("./ixia/" + filename)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// copy source file to destination file
	bar := progressbar.DefaultBytes(
		srcFileInfo.Size(),
		"uploading",
	)

	bytes, err := io.Copy(io.MultiWriter(dstFile, bar), srcFile)
	if err != nil {
		return err
	}
	fmt.Printf("%d bytes copied\n", bytes)
	return nil
}

func upgradeIxOS() {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	//connect SSH to the chassis
	fmt.Printf("\nConnecting to chassis at: %v:22\n\n", host)
	fmt.Fprintf(output, "\nConnecting to chassis at: %v:22\n\n", host)
	conn, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		fmt.Printf("Failed to connect: %v", err)
		fmt.Fprintf(output, "Failed to connect: %v", err)
		return
	}
	defer conn.Close()

	//todo: confirm the active version is lower than the target version
	//  command: show chassis active
	//  then parse the response and filename for version validation

	//trigger the cli
	res := runCommand(conn, "")

	//get main prompt
	fmt.Println("Checking chassis current version")
	fmt.Fprintln(output, "Checking chassis current version")
	res = runCommand(conn, "enter chassis")
	//does it list the target version?
	if version == "" {
		version = offlineUpdatePackage[22 : len(offlineUpdatePackage)-17]
	}

	if bytes.Contains(res.Bytes(), []byte(version)) == true {
		fmt.Println("Chassis current version already matches ", version)
		fmt.Fprintln(output, "Chassis current version already matches ", version)
		//exit as we dont need to do anything
		return
	}

	//check for online updates avail
	fmt.Println("Checking chassis for available updates")
	res = runCommands(conn, "enter chassis", "show ixos available-updates")
	resString := res.String()
	resString = resString[len(resString)-150 : len(resString)] //just grab the last part of the response

	//does it include the target version?
	if bytes.Contains(res.Bytes(), []byte("No updates available.")) == true || bytes.Contains([]byte(resString), []byte(version)) == false {

		//check the provided update package flag
		if offlineUpdatePackage == "" {
			fmt.Println("\nNo online update avaible to the chassis,")
			fmt.Println("configure DNS and ensure the chassis has an internet connection or")
			fmt.Println("rerun updateIxOS withthe --package option.")
			return
		}

		fmt.Println("\nNo online option availible on chassis,")
		fmt.Println("configure DNS and ensure the chassis has an")
		fmt.Println("internet connection to use this feature.")

		//If the update target file is not in the availible list
		// create new SFTP client
		client, err := sftp.NewClient(conn)
		if err != nil {
			log.Fatal(err)
		}
		defer client.Close()

		//upload the update package
		fmt.Printf("\nUploading the ./%s, update file...\n", offlineUpdatePackage)
		err = uploadFile(client, offlineUpdatePackage)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Upload complete")

	} else if bytes.Contains([]byte(resString), []byte(version)) == true {
		//since the update is already availible there is no need to upload
		//
		fmt.Println("Online update availible, trying install")
	}
	//run the install
	fmt.Println("run install ixos here")
	_ = runCommands(conn, "enter chassis", "install ixos "+version)

}

//loop is the gio window for a local os gui
func loop(w *app.Window) {
	th := material.NewTheme(gofont.Collection())
	var ops op.Ops

	// UI state.
	var (
		upgrade      widget.Clickable
		close        widget.Clickable
		hostField    = &widget.Editor{SingleLine: true, Submit: true}
		userField    = &widget.Editor{SingleLine: true, Submit: true}
		passField    = &widget.Editor{SingleLine: true, Submit: true}
		fileField    = &widget.Editor{SingleLine: true, Submit: true}
		versionField = &widget.Editor{SingleLine: true, Submit: true}
	)

	//loop watching for events
	for {
		select {

		case e := <-w.Events():
			if _, ok := e.(system.DestroyEvent); ok {
				os.Exit(0)
			}
			if e, ok := e.(system.FrameEvent); ok {
				gtx := layout.NewContext(&ops, e)
				//sz := image.Point{X: 400, Y: 400}

				for upgrade.Clicked() {
					//clear the output buffer
					output.Reset()

					//post any fields that are active
					if hostField.Text() != "" {
						host = hostField.Text()
					}
					if versionField.Text() != "" {
						version = versionField.Text()
					}
					if fileField.Text() != "" {
						offlineUpdatePackage = fileField.Text()
					}
					if userField.Text() != "" {
						username = userField.Text()
					}
					if passField.Text() != "" {
						password = passField.Text()
					}

					//validate the input fields
					if host == "" {
						fmt.Fprintln(output, "You must provide a hostname or IP adress for the target Chassis.")
						break
					}
					if version == "" && offlineUpdatePackage == "" {
						fmt.Fprintln(output, "You must provide a local upgrade filename, or a target version for online updates")
						break
					}

					fmt.Fprintln(output, "running upgrade..")
					go upgradeIxOS()
				}
				for close.Clicked() {
					w.Close()
				}
				//if its a Field event
				for _, e := range hostField.Events() {
					if e, ok := e.(widget.SubmitEvent); ok {
						host = e.Text
						hostField.SetText(host)
					}
				}
				for _, e := range userField.Events() {
					if e, ok := e.(widget.SubmitEvent); ok {
						username = e.Text
						userField.SetText(username)
					}
				}
				for _, e := range passField.Events() {
					if e, ok := e.(widget.SubmitEvent); ok {
						password = e.Text
						passField.SetText(password)
					}
				}
				for _, e := range versionField.Events() {
					if e, ok := e.(widget.SubmitEvent); ok {
						version = e.Text
						versionField.SetText(version)
					}
				}
				for _, e := range fileField.Events() {
					if e, ok := e.(widget.SubmitEvent); ok {
						offlineUpdatePackage = e.Text
						fileField.SetText(offlineUpdatePackage)
						version = offlineUpdatePackage[22 : len(offlineUpdatePackage)-17]
						versionField.SetText(version)
					}
				}
				layout.W.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.H4(th, "Chassis Information").Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Px(unit.Dp(300))
							gtx.Constraints.Max.X = gtx.Px(unit.Dp(300))
							e := material.Editor(th, hostField, "Hostname or IP")
							border := widget.Border{Color: color.RGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Px(4)}
							return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Px(unit.Dp(300))
							gtx.Constraints.Max.X = gtx.Px(unit.Dp(300))
							e := material.Editor(th, userField, username)
							border := widget.Border{Color: color.RGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Px(4)}
							return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Px(unit.Dp(300))
							gtx.Constraints.Max.X = gtx.Px(unit.Dp(300))
							e := material.Editor(th, passField, password)
							border := widget.Border{Color: color.RGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Px(4)}
							return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return material.H4(th, "Upgrade Information").Layout(gtx)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Px(unit.Dp(300))
							gtx.Constraints.Max.X = gtx.Px(unit.Dp(300))
							e := material.Editor(th, fileField, "Offline Package Filename")
							border := widget.Border{Color: color.RGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Px(2)}
							return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Px(unit.Dp(300))
							gtx.Constraints.Max.X = gtx.Px(unit.Dp(300))
							e := material.Editor(th, versionField, "Target Version")
							border := widget.Border{Color: color.RGBA{A: 0xff}, CornerRadius: unit.Dp(8), Width: unit.Px(2)}
							return border.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								//gtx.Constraints.Width.Min = unit.Dp(200)
								return layout.UniformInset(unit.Dp(8)).Layout(gtx, e.Layout)
							})
						}),
						//the output block
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							gtx.Constraints.Min.X = gtx.Px(unit.Dp(400))
							//txt := fmt.Sprintf()
							return material.Body1(th, output.String()).Layout(gtx)

						}),
						//the buttons
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Alignment: layout.Middle,
							}.Layout(gtx,
								rigidInset(material.Button(th, &upgrade, "Upgrade").Layout),
								rigidInset(material.Button(th, &close, "Close").Layout),
							)
						}),
					)
				})

				e.Frame(gtx.Ops)
			}
		}

	}
}
func rigidInset(w layout.Widget) layout.FlexChild {
	return layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Sp(5)).Layout(gtx, w)
	})
}

func main() {

	//command line flags
	flag.StringVar(&host, "host", "", "Native IxOS Chassis managment IP Address (required)")
	flag.StringVar(&username, "user", "admin", "Username for host (default: admin)")
	flag.StringVar(&password, "password", "admin", "Password for username (default: admin)")
	flag.StringVar(&offlineUpdatePackage, "package", "", "Native IxOS Offline update package")
	flag.StringVar(&version, "version", "", "Native IxOS Version to use as an upgrade target (required if no package supplied)")
	flag.Parse()

	//check required input
	if host == "" {
		fmt.Println("Missing arguments")
		fmt.Println("Usage example: updateIxOS.exe --host 192.168.1.1 --version 9.10.2000.11")
		//launch the gui version
		fmt.Println("launching GUI version")

		w := app.NewWindow(
			app.Title("IxOS Updater Utility"),
			app.Size(
				unit.Dp(400),  //width
				unit.Dp(400)), //height
		)
		loop(w)

		app.Main()
		os.Exit(0)
	}

	//need at min a target version, can also be a package
	if version == "" && offlineUpdatePackage == "" {
		fmt.Println("Missing arguments")
		fmt.Println("The --package [IxOS Offline Package] or --version [IxOS build number] argument is required.\nPlease use upgrade --help for more information")
		fmt.Println("    Usage example: updateIxOS.exe --host 192.168.1.1 --version 9.10.2000.11")
		return
	} else if offlineUpdatePackage != "" && bytes.Contains([]byte(offlineUpdatePackage), []byte(".tar.gz.gpg")) == false {
		fmt.Println("The --package [IxOS Offline Package] argument is invalid.\n  Please use a .tar.gz.gpg valid file from https://support.ixiacom.com")
		fmt.Println("    Usage example: updateIxOS.exe --host 192.168.1.1 --package Ixia_Hardware_Chassis_9.10.3000.11-IxOS.tar.gz.gpg")
		return
	}

	//run the upgrade
	upgradeIxOS()

	os.Exit(0)
}
