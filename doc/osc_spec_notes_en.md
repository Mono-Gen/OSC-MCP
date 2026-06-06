# OSC (Open Sound Control) 1.0 Specification Notes

This technical reference serves as a guideline for parsing and encoding OSC packets accurately, preventing common programming errors regarding byte padding and alignment.

---

## 1. Data Representation & 4-Byte Alignment

All data elements inside an OSC packet must be **aligned to a 4-byte (32-bit) boundary**. If the size of an element is not a multiple of 4 bytes, it must be padded with trailing null bytes (`0x00`).

### 1.1. OSC-string
- ASCII characters followed by a null terminator (`0x00`) plus 0 to 3 extra null bytes to align the total length to a multiple of 4.
- **Important**: An OSC-string must **always contain at least one null terminator** at the end.
- **Examples**:
  - `"OSC"` (3 chars) $\rightarrow$ `"OSC"` (3B) + `\x00` (1B) = `4 bytes` (No extra padding)
  - `"DATA"` (4 chars) $\rightarrow$ `"DATA"` (4B) + `\x00` (1B) + `\x00\x00` (2B) = `8 bytes` (3 null bytes total)

### 1.2. OSC-blob
- An `int32` size header + raw binary bytes + 0 to 3 null padding bytes.
- The total payload length (excluding the size header) must be a multiple of 4.

### 1.3. Numeric Types
- **`int32` (i)**: 32-bit two's complement signed integer (Big-Endian / Network Byte Order).
- **`float32` (f)**: 32-bit IEEE 754 single-precision float (Big-Endian).

---

## 2. OSC Message Structure

An OSC Message consists of:
```
+-----------------------------------------------------------+
| OSC Address Pattern (OSC-string starting with '/')       |
+-----------------------------------------------------------+
| OSC Type Tag String (OSC-string starting with ',')       |
+-----------------------------------------------------------+
| OSC Arguments (4-byte aligned binary payloads)            |
+-----------------------------------------------------------+
```

### 2.1. OSC Type Tags (OSC 1.0)
- `i`: 32-bit Integer
- `f`: 32-bit Float
- `s`: OSC-string
- `b`: OSC-blob
- `T`: True (No payload data)
- `F`: False (No payload data)
- `N`: Null (No payload data)
- `I`: Impulse/Infinitum (No payload data)

---

## 3. OSC Bundle Structure

A Bundle encapsulates multiple OSC Messages or nested Bundles to execute simultaneously under a single time tag.

```
+-----------------------------------------------------------+
| "#bundle" (OSC-string) -> ("#bundle\x00")                 |
+-----------------------------------------------------------+
| OSC Time Tag (64-bit NTP timestamp, Big-Endian)           |
+-----------------------------------------------------------+
| Element 1 Size (int32)                                    |
+-----------------------------------------------------------+
| Element 1 Payload (OSC Message or OSC Bundle)             |
+-----------------------------------------------------------+
```

- **OSC Time Tag**: 64-bit NTP timestamp. The value `0x0000000000000001` indicates immediate execution.
- **Bundle Element**: Pairs of an `int32` size header and the raw payload of that size.

---

## 4. Go Implementation Guidelines

1. **Big-Endian Handling**:
   - Utilize `binary.BigEndian.Uint32` and `binary.BigEndian.PutUint32` to encode/decode integers and float bits.
2. **Defensive Boundary Checks**:
   - Verify remaining slice boundaries before performing indexing operations (`data[offset:offset+4]`) to prevent runtime panic errors.
3. **UDP Buffer Boundaries**:
   - Read incoming packets into a `65535` byte buffer to avoid truncation. Reuse the buffer via `sync.Pool`.
4. **Thread Safety**:
   - Protect maps and caches using `sync.RWMutex` since UDP listener goroutines and MCP threads access shared states concurrently.
5. **stdio Stream Safety**:
   - Strictly write all logging and debug details to `os.Stderr`. Do not write arbitrary messages to `os.Stdout`.
6. **Argument Range Auditing**:
   - Audit `float64` numbers decoded from JSON before casting them to `int32` to verify they do not overflow.
