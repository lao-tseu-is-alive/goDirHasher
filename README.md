# **goDirHasher**
goDirHasher allows you to calculate a unique digital fingerprints (SHA-256) for all files
in your folders to help verify their integrity and identify duplicates.

A fast and efficient command-line utility written in Go for calculating
and verifying SHA256 hashes of files, including recursive directory processing.
Inspired by the standard sha256sum command, goDirHasher leverages Go's concurrency
and hashing features and optimized hashing libraries for performance.

## **🚀 Performance**
Testing this with a set of 70'000 files for a 35 GigaByte.

 **6.3 seconds with goDirHasher, versus 78.5 seconds** 
 with the standard sha256sum command. 
 
**That's 12 times faster !**

```bash
  cgil@pulsar:~/cgdev/golang/goDirHasher$ du -hs Godoc/
  38G	Godoc/
  
  cgil@pulsar:~/cgdev/golang/goDirHasher$ /usr/bin/time -f "%es" ./goDirHasher-linux-amd64 -workers 25 -o SHA256.txt Godoc/*
  2025/05/09 11:26:52 🚀🚀 Starting App:'goDirHasher', ver:0.1.0, from: https://github.com/lao-tseu-is-alive/goDirHasher
  ℹ️ Using maxWorkers = 25 
  🔢 Entering calculate mode...
  ℹ️ Found 70000 files to process.
  ℹ️ Writing output to file: SHA256.txt
  ✅ Successfully calculated hashes for 70000 files.
  6.35s 
    
  cgil@pulsar-2021:~/cgdev/golang/goDirHasher$ /usr/bin/time -f "%es " ./goDirHasher -workers 25 -c SHA256.txt 
  2025/05/09 11:30:26 🚀🚀 Starting App:'goDirHasher', ver:0.1.0, from: https://github.com/lao-tseu-is-alive/goDirHasher
  ℹ️ Using maxWorkers = 25 
  🕵️ Entering check mode...
  🏴󠁲󠁯󠁩󠁦󠁿 Checking if hash file exists: SHA256.txt
  ✅ Opening hash file: SHA256.txt
  ✅ Successfully parsed 70000 entries from SHA256.txt.
  ✅ 70000 files processed, 70000 valid, 0 invalid.
  6.26s 
  
  cgil@pulsar:~/cgdev/golang/goDirHasher$ /usr/bin/time -f "%es " sha256sum --quiet -c SHA256.txt 
  78.51s 
```

## **✨ Features**

* **Calculate SHA256 Hashes:** Generate SHA256 hashes for single files, multiple files, or all files within specified directories (recursively).
* **Verify SHA256 Hashes:** Check files against a list of known hashes (in sha256sum format) from a file or standard input.
* **Concurrency:** Utilizes a worker pool to process files concurrently, significantly speeding up operations on multi-core processors.
* **Optimized Hashing:** Uses the github.com/minio/sha256-simd library for potentially faster hashing on supported architectures.
* **Standard Format:** Outputs hashes in the widely compatible sha256sum format (hash filepath).
* **Profiling:** Built-in support for CPU and memory profiling to help identify performance bottlenecks.

## **🛠️ Installation**
### **Download from github**
You can just download the latest release for your operating system from [GitHub releases](https://github.com/lao-tseu-is-alive/goDirHasher/releases)
### **Using go get**
To install goDirHasher, make sure you have Go installed on your system. Then, you can use go install:

go install github.com/lao-tseu-is-alive/goDirHasher@latest

This will install the goDirHasher executable in your $GOPATH/bin directory. Make sure this directory is in your system's PATH.

Alternatively, you can clone the repository and build it manually:

git clone https://github.com/lao-tseu-is-alive/goDirHasher.git  
cd goDirHasher  
go build

This will create the goDirHasher executable in the current directory.

## **📖 Usage**

goDirHasher operates in two main modes: **calculate** (default) and **check** (-c).

goDirHasher \[OPTIONS\] \[FILE...\]

### **Calculate Mode (Default)**

When no \-c flag is provided, goDirHasher calculates and outputs the SHA256 hashes for the specified files or directories. If a directory is provided, it will recursively find and hash all files within it.

* **Calculate hash for a single file:**  
  goDirHasher myfile.txt

* **Calculate hashes for multiple files:**  
  goDirHasher file1.txt /path/to/another/file.dat

* **Calculate hashes for all files in the current directory (and subdirectories):**  
  goDirHasher .

* **Calculate hashes for a specific directory:**  
  goDirHasher /path/to/my/directory

* **Save calculated hashes to a file:**  
  goDirHasher /path/to/my/directory \> hashes.txt  
  \# Or using the \-o flag  
  goDirHasher \-o hashes.txt /path/to/my/directory

### **Check Mode (-c)**

Use the \-c flag to verify files against a list of hashes. The input should be a file (or standard input) in the sha256sum format (hash filepath).

* **Check hashes from a file:**  
  goDirHasher \-c hashes.txt

* **Check hashes from standard input:**  
  cat hashes.txt | goDirHasher \-c \-

  *(Using \- as the file argument explicitly tells goDirHasher to read from stdin)*

goDirHasher will output FAILED for any file whose calculated hash does not match the hash in the input file. By default, it only reports failures. It will exit with a non-zero status code if any checks fail.

### **Options**

* \-c: Enable check mode. Verify files against a list of hashes.
* \-o string: Output file for calculated hashes (defaults to stdout).
* \-workers int: Number of concurrent workers to use (default 15, max 50). Adjust this based on your system's capabilities and the type of storage you are reading from.
* \-cpuprofile string: Write CPU profile to the specified file.
* \-memprofile string: Write memory profile to the specified file.

## **📊 Profiling**

You can use the \-cpuprofile and \-memprofile flags to generate profiling data. This data can be analyzed using Go's built-in pprof tool to understand the performance characteristics of goDirHasher and identify areas for optimization.

goDirHasher \-cpuprofile cpu.prof \-memprofile mem.prof .  
go tool pprof cpu.prof  
go tool pprof mem.prof

## **👋 Contributing**

Contributions are welcome\! Please feel free to open issues or submit pull requests.

## **📄 License**

This project is licensed under the MIT License \- see the [LICENSE](http://docs.google.com/LICENSE) file for details.