env
===

This is a copy of the `flag` package adapted to work with environment variables.

Use this package to:

- Define different environment variables, their default value, and description.
- Define different environments, e.g. one set of environment variables for each
  subcommand.
- Get a description of the environment variables defined using the `-h/--help`
  command line flag parsing behavior.
- Get a description of the environment variables defined by setting `HELP` or
  `H`.
- Add support for your types to be used as environment variables.
