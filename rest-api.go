package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/keltia/archive"
)

func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
					panic(err)
			}
	}
}

func enableCors(w *http.ResponseWriter) {
	//(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

    var filenames []string

    r, err := zip.OpenReader(src)
    if err != nil {
        return filenames, err
    }
    defer r.Close()

    for _, f := range r.File {

        // Store filename/path for returning and using later on
        fpath := filepath.Join(dest, f.Name)

        // Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
        if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
            return filenames, fmt.Errorf("%s: illegal file path", fpath)
        }

        filenames = append(filenames, fpath)

        if f.FileInfo().IsDir() {
            // Make Folder
            os.MkdirAll(fpath, os.ModePerm)
            continue
        }

        // Make File
        if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
            return filenames, err
        }

        outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
        if err != nil {
            return filenames, err
        }

        rc, err := f.Open()
        if err != nil {
            return filenames, err
        }

        _, err = io.Copy(outFile, rc)

        // Close the file without defer to close before next iteration of loop
        outFile.Close()
        rc.Close()

        if err != nil {
            return filenames, err
        }
    }
    return filenames, nil
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) ([]string, error) {

    var filenames []string

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return filenames, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return filenames, nil

		// return any other error
		case err != nil:
			return filenames, err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

        filenames = append(filenames, target)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return filenames, err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return filenames, err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return filenames, err
			}
			
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}

	return filenames, nil
}

func fireDMARCProcessor(thefilepath string, args []string) string {
	//Check to make sure this is an XML file...

	if strings.Contains(thefilepath, ".xml") {
		fmt.Println("Processing DMARC report: " + thefilepath)

		ctx, err := Setup(args)
		if ctx == nil {
			fmt.Println(err)
		}

		var txt string

		file := thefilepath

		fmt.Println("Analyzing " + file)

		if filepath.Ext(file) == ".zip" ||
			filepath.Ext(file) == ".ZIP" {

			txt, err = HandleZipFile(ctx, file)
			if err != nil {
				fmt.Println("txt")
				fmt.Println(err)
			}
		} else {
			in, err := SelectInput(file)
			if err != nil {
				fmt.Println("in")
				fmt.Println(err)
			}
			defer in.Close()

			typ := archive.Ext2Type(fType)

			txt, err = HandleSingleFileJSON(ctx, in, typ)
			if err != nil {
				fmt.Println("handle")
				fmt.Println(err)
			}
		}
		fmt.Println(txt)

		return txt

	} else {
		fmt.Println("Failed opening " + thefilepath + ", NOT VALID FILE TYPE")

		return `{"status":"failed", "stage": "fireDMARCProcessor - File type check"}`
	}
	
}

func uploadFile(w http.ResponseWriter, r *http.Request) {

	enableCors(&w)

    fmt.Println("File Upload Endpoint Hit")

    // Parse our multipart form, 10 << 20 specifies a maximum
    // upload of 10 MB files.
    r.ParseMultipartForm(10 << 20)
    // FormFile returns the first file for the given key `myFile`
    // it also returns the FileHeader so we can get the Filename,
    // the Header and the size of the file
    file, handler, err := r.FormFile("bundleFile")
    if err != nil {
        fmt.Println("Error Retrieving the File")
        fmt.Println(err)
    }
	defer file.Close()
	
	bundleFileExt := filepath.Ext(handler.Filename)
	bundlefileName := handler.Filename;

    fmt.Printf("Uploaded File: %+v\n", bundlefileName)
    fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)
	fmt.Printf("File Extension: %+v\n", bundleFileExt)

    // Create a temporary file within our temp-images directory that follows
	// a particular naming pattern

	CreateDirIfNotExist(os.TempDir() + "/temp-bundles")
	CreateDirIfNotExist(os.TempDir() + "/temp-bundles/extracts")

    tempFile, err := ioutil.TempFile(os.TempDir() + "/temp-bundles", "upload-*" + bundleFileExt)
    if err != nil {
		fmt.Println(err)
    }
    defer tempFile.Close()

    // read all of the contents of our uploaded file into a
    // byte array
    fileBytes, err := ioutil.ReadAll(file)
    if err != nil {
        fmt.Println(err)
    }
    // write this byte array to our temporary file
    tempFile.Write(fileBytes)
    // return that we have successfully uploaded our file!
	fmt.Println("Successfully Uploaded File")

	var processorResults string

	// Determine file type
	switch bundleFileExt {
		case ".zip":
			fmt.Println("File type is ZIP, extracting...")
			files, err := Unzip(tempFile.Name(), os.TempDir() + "/temp-bundles/extracts")
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))

			//For every XML file, run the processor...
			for _, file := range files {
				if strings.Contains(file, ".xml") {
					processorResults = fireDMARCProcessor(file, flag.Args())

					os.Remove(file)
				}
			}

		case ".gz":
			fmt.Println("File type is Gunzip, extracting " + tempFile.Name() + "...")
			fileReader, err := os.Open(tempFile.Name())
			if err != nil {
				fmt.Println(err)
			}
			defer fileReader.Close()

			fz, err := gzip.NewReader(fileReader)
			if err != nil {
				fmt.Println(err)
			}
			defer fz.Close()

			s, err := ioutil.ReadAll(fz)
			if err != nil {
				fmt.Println(err)
			}

			reformattedFilename := strings.Replace(bundlefileName, "." + bundleFileExt, "", -1)
			reformattedFullPath := os.TempDir() + "/temp-bundles/" + reformattedFilename

			ferr := ioutil.WriteFile(reformattedFullPath, s, 0644)
			if ferr != nil {
				fmt.Println(ferr)
			}

			fmt.Println("File is gz XML, processing...")
			fmt.Println("Sending file " + reformattedFullPath)
			processorResults = fireDMARCProcessor(reformattedFullPath, flag.Args())

			os.Remove(reformattedFullPath)


		case ".xml":
			fmt.Println("File is raw XML, proceeding...")
			processorResults = fireDMARCProcessor(tempFile.Name(), flag.Args())

		default:
			fmt.Printf("Failed to open file type %s.\n", bundleFileExt)
	}

	os.Remove(tempFile.Name())

	fmt.Fprintf(w, processorResults + "\n")
	
}

func healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Health endpoint hit")
	fmt.Fprintf(w, "ok")
}

func setupRoutes() {
	http.HandleFunc("/api/v1/upload_bundle", uploadFile)
	http.HandleFunc("/healthz", healthz)
    http.ListenAndServe(":8080", nil)
}