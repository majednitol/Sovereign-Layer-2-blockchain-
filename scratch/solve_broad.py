import sys
import itertools

CHARSET = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

def bech32_polymod(values):
    generator = [0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3]
    chk = 1
    for value in values:
        top = chk >> 25
        chk = ((chk & 0x1ffffff) << 5) ^ value
        for i in range(5):
            chk ^= generator[i] if ((top >> i) & 1) else 0
    return chk

def bech32_hrp_expand(hrp):
    return [ord(x) >> 5 for x in hrp] + [0] + [ord(x) & 31 for x in hrp]

def create_checksum(hrp, data, spec):
    values = bech32_hrp_expand(hrp) + data
    const = 0x2bc830a3 if spec == "bech32m" else 1
    polymod = bech32_polymod(values + [0, 0, 0, 0, 0, 0]) ^ const
    return [(polymod >> 5 * (5 - i)) & 31 for i in range(6)]

# Decoded data 5-bit array from 'cosmos1n4agesyhv32aw03zu3xlsemvc3dvq4d635c5mq'
data_str = "n4agesyhv32aw03zu3xlsemvc3dvq4d6"
data_5bit = [CHARSET.index(c) for c in data_str]

target_suffix = "uccuw0"
target_checksum = [CHARSET.index(c) for c in target_suffix]

# Let's search prefixes of length 5 to 10
# HRP is composed of letters 'a'-'z'
found = False
for length in range(5, 11):
    print(f"Searching HRP of length {length}...")
    if length == 5:
        for p in itertools.product(range(97, 123), repeat=5):
            hrp = "".join(chr(c) for c in p)
            checksum = create_checksum(hrp, data_5bit, "bech32")
            if checksum == target_checksum:
                print(f"FOUND MATCH: {hrp} (bech32)")
                found = True
            checksum_m = create_checksum(hrp, data_5bit, "bech32m")
            if checksum_m == target_checksum:
                print(f"FOUND MATCH: {hrp} (bech32m)")
                found = True
    elif length == 6:
        # Check specific starts or do optimized loop
        # Since we know it expected "uccuw0" for "cosmos1..." when passed "cosmos1...", wait!
        # If the input was "cosmos1...", the prefix parsed by the decoder was "cosmos"!
        # And the data part was "n4agesyhv32aw03zu3xlsemvc3dvq4d6".
        # So the input HRP was DEFINITELY "cosmos"!
        # But why did the verify function calculate the expected checksum as "uccuw0" instead of "35c5mq"???
        # Ah!!!
        # Could the encoding spec used by the verify function be Bech32m?
        # But we saw Bech32m for "cosmos" is "yggc7z"!
        # Could the HRP used inside the contract's addr_validate be DIFFERENT?
        # Yes! Inside the contract, the SDK uses the prefix defined by the contract's build or imports, or the chain node's custom SDK config!
        # But wait! If the chain node's custom SDK config uses a different HRP (like "sovereign"), then when the contract calls addr_validate, the VM's api.addr_validate parses the string "cosmos1n4agesyhv32aw03zu3xlsemvc3dvq4d635c5mq".
        # And since it has prefix "cosmos", but the VM expects prefix "sovereign", it returns "invalid prefix" or "invalid checksum"!
        # In Cosmos SDK, Bech32 decoding parses the HRP from the string itself!
        # So it decoded the HRP as "cosmos".
        # But then it checked the checksum of "cosmos1..." using Bech32/Bech32m.
        # Wait! If HRP is "cosmos", and it checks the checksum using Bech32, it should match "35c5mq"!
        # Why did it expect "uccuw0"?
        # Let's check length 6 HRPs to see if "cosmos" is NOT the HRP that matches "uccuw0"!
        # Let's search all length 6 HRPs:
        for p in itertools.product(range(97, 123), repeat=6):
            hrp = "".join(chr(c) for c in p)
            # Only test if it ends with 's' or starts with 'c' to speed up, or do all?
            # Let's do all length 6 HRPs starting with 'c':
            if p[0] != 99: # 'c'
                continue
            checksum = create_checksum(hrp, data_5bit, "bech32")
            if checksum == target_checksum:
                print(f"FOUND MATCH (len 6): {hrp} (bech32)")
            checksum_m = create_checksum(hrp, data_5bit, "bech32m")
            if checksum_m == target_checksum:
                print(f"FOUND MATCH (len 6): {hrp} (bech32m)")
        break
