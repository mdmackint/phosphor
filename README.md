# Phosphor

Phosphor is a small web server written in Go with very few dependencies, for searching through your library catalogue.

## Quick start

As long as you have Go installed, setup is very easy. First, however, you need a compatible catalogue in CSV format. Phosphor is compatible with the CSV files exported from [Libib](https://libib.com).

Once you have your catalogue, save it as `catalogue.csv` in the same folder as Phosphor is downloaded to.

> Note that the catalogue is embedded within the binary *at compile time*. You will need to recompile the binary after updating the catalogue.

Then, run `sudo go run .` to start the Phosphor server. Root priveliges are required, because the server binds to port 80 which requires root on Linux.

That should be everything.