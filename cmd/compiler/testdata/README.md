# Test scripts

For end-to-end testing of the compiler, the files in testdata/scripts represent full test runs with inputs and outputs.

The file format is a `txtar` archive, with a MIME style header to set parameters for the test.

The files in the archive will be available to the compiler as if it was running in that directory. 