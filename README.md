# Overview

MiniSQL is a lightweight relational database built entirely in Go, focused on implementing and understanding core database internals such as storage management and transaction handling. 

> Query processing is planned and listed as an upcoming feature.

## Table of Contents

- [Installation](#installation)
- [Features](#features)
- [Upcoming Features](#upcoming-features)
- [Supported Commands](#supported-commands)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Prerequisites

- [Golang](https://golang.org/dl/) (version 1.23 or later)
- [Git](https://git-scm.com/downloads)

### Clone the Repository

Clone the repository to your local machine:

```bash
git clone https://github.com/rahul-ringane/MiniSQL.git && cd MiniSQL
```

### Build

Build the project using Go:

```bash
go mod tidy
go build -o minisql
```

### Run

To start the MiniSQL server, execute:

```bash
./minisql
```

## Features

- **B+ Tree Storage Engine with Indexing Support**: Enables fast data retrieval, which is critical for database performance, especially in scenarios involving large datasets.

- **Free List Management for Node Reuse**: The database manages a free list to reuse nodes, which is a strategy to optimize storage usage by recycling space from freed nodes. This helps reduce fragmentation and improve disk space efficiency.

- **Transaction Support**: MiniSQL supports transactions, ensuring data consistency and integrity through atomic operations.
- **Concurrent Reads**: The ability to handle concurrent reads enhances performance by allowing multiple users to read data simultaneously without locking issues, making it suitable for read-heavy applications.

## Upcoming Features

- **Query Processing**: Enhancing MiniSQL with query capabilities to support more complex data retrieval and manipulation.

- **Bug Fixes**: Ongoing efforts to address and resolve identified bugs to improve stability and reliability.

## Supported Commands

- **CREATE**
- **INSERT**
- **GET**
- **UPDATE**
- **DELETE**
- **BEGIN**
- **COMMIT**
- **ABORT**

## Contributing

Contributions are welcome! If you'd like to contribute to MiniSQL, please follow these steps:

1.  **Fork the repository.**

2.  **Create a new branch:** `git checkout -b feature/YourFeature`

3.  **Commit your changes:** `git commit -am "Add new feature"`

4.  **Push to the branch:** `git push origin feature/YourFeature`

5.  **Open a Pull Request** describing your changes and the problem they solve.