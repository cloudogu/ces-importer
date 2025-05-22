// Package sync provides functions to copy data from the exporter system to the importer system (where this application
// is running).
//
// While there are two fundamentally different ways to achieve this synchronization (regular sync with cron style and
// final sync based on a fixed timestamp), the technical level keeps being the same -- only the timing makes the
// difference between these two.
package sync
