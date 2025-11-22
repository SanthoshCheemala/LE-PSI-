#!/usr/bin/env python3
"""
Synthetic Dataset Generator for PSI-based Sanctions Screening
Generates:
1. Server dataset: Malicious/sanctioned accounts
2. Client dataset: Bank customers (with controlled overlap for testing)
"""

import json
import csv
import random
import hashlib
from datetime import datetime, timedelta
from faker import Faker
import argparse

# Initialize Faker with seed for reproducibility
fake = Faker()
Faker.seed(42)
random.seed(42)

# Sanctions program categories
SANCTION_PROGRAMS = [
    "OFAC_SDN",
    "EU_SANCTIONS", 
    "UN_SECURITY_COUNCIL",
    "TERRORISM_FINANCING",
    "NARCOTICS_TRAFFICKING",
    "WEAPONS_PROLIFERATION",
    "CYBER_CRIMES",
    "MONEY_LAUNDERING",
    "HUMAN_TRAFFICKING",
    "CORRUPTION"
]

# Risk levels
RISK_LEVELS = ["CRITICAL", "HIGH", "MEDIUM", "LOW"]

# Geographic regions with higher fraud rates
HIGH_RISK_COUNTRIES = [
    "RU", "KP", "IR", "SY", "VE", "BY", "MM", "AF", "IQ", "LY",
    "SO", "SD", "YE", "ZW", "CU", "NI"
]

MEDIUM_RISK_COUNTRIES = [
    "CN", "PK", "NG", "BD", "PH", "ID", "UA", "EG", "TR", "VN"
]

LOW_RISK_COUNTRIES = [
    "US", "GB", "CA", "AU", "DE", "FR", "JP", "SE", "CH", "NL",
    "NO", "DK", "FI", "SG", "NZ", "IE", "AT", "BE", "ES", "IT"
]


def generate_psi_key(name, dob, country):
    """Generate consistent PSI key for matching"""
    normalized_name = name.lower().strip()
    normalized_dob = dob if dob else ""
    normalized_country = country.upper() if country else ""
    return f"{normalized_name}|{normalized_dob}|{normalized_country}"


def generate_hash(psi_key):
    """Generate SHA-256 hash for PSI key"""
    return hashlib.sha256(psi_key.encode('utf-8')).hexdigest()


def generate_sanctioned_entity(entity_id, risk_distribution):
    """Generate a single sanctioned entity with realistic attributes"""
    
    # Determine risk level based on distribution
    risk = random.choices(
        RISK_LEVELS,
        weights=risk_distribution,
        k=1
    )[0]
    
    # Select country based on risk
    if risk in ["CRITICAL", "HIGH"]:
        country = random.choice(HIGH_RISK_COUNTRIES)
    elif risk == "MEDIUM":
        country = random.choice(MEDIUM_RISK_COUNTRIES)
    else:
        country = random.choice(LOW_RISK_COUNTRIES)
    
    # Generate person details
    name = fake.name()
    dob = fake.date_of_birth(minimum_age=20, maximum_age=80).strftime("%Y-%m-%d")
    
    # Generate aliases (sanctioned entities often have multiple names)
    num_aliases = random.choices([0, 1, 2, 3], weights=[0.4, 0.3, 0.2, 0.1], k=1)[0]
    aliases = [fake.name() for _ in range(num_aliases)]
    
    # Sanction details
    program = random.choice(SANCTION_PROGRAMS)
    sanction_date = fake.date_between(start_date='-10y', end_date='today').strftime("%Y-%m-%d")
    
    # Additional identifiers
    passport = fake.bothify(text='??#######').upper() if random.random() > 0.3 else None
    national_id = fake.bothify(text='##########') if random.random() > 0.4 else None
    
    # Generate PSI key
    psi_key = generate_psi_key(name, dob, country)
    psi_hash = generate_hash(psi_key)
    
    return {
        "entity_id": f"SANC_{entity_id:06d}",
        "name": name,
        "aliases": aliases,
        "dob": dob,
        "country": country,
        "risk_level": risk,
        "sanction_program": program,
        "sanction_date": sanction_date,
        "passport_number": passport,
        "national_id": national_id,
        "psi_key": psi_key,
        "psi_hash": psi_hash,
        "last_updated": datetime.now().strftime("%Y-%m-%d")
    }


def generate_customer(customer_id, is_match=False, sanctioned_entity=None):
    """Generate a customer record (with optional match to sanctioned entity)"""
    
    if is_match and sanctioned_entity:
        # Create a customer that matches a sanctioned entity
        # Use same name, dob, country but different customer ID
        name = sanctioned_entity["name"]
        dob = sanctioned_entity["dob"]
        country = sanctioned_entity["country"]
        
        # Small chance of slight variation (fuzzy match scenario)
        if random.random() < 0.1:
            # Add middle initial or suffix
            name = name + " " + random.choice(["Jr.", "Sr.", "II", "III"])
    else:
        # Generate clean customer
        country = random.choice(LOW_RISK_COUNTRIES + MEDIUM_RISK_COUNTRIES)
        name = fake.name()
        dob = fake.date_of_birth(minimum_age=18, maximum_age=75).strftime("%Y-%m-%d")
    
    # Customer-specific fields
    account_number = fake.iban()
    email = fake.email()
    phone = fake.phone_number()
    account_balance = round(random.uniform(100, 500000), 2)
    account_opened = fake.date_between(start_date='-5y', end_date='today').strftime("%Y-%m-%d")
    
    # Transaction behavior
    monthly_transactions = random.randint(5, 200)
    avg_transaction = round(random.uniform(50, 10000), 2)
    
    # Address
    address = fake.address().replace('\n', ', ')
    
    # Generate PSI key
    psi_key = generate_psi_key(name, dob, country)
    psi_hash = generate_hash(psi_key)
    
    return {
        "customer_id": f"CUST_{customer_id:06d}",
        "name": name,
        "dob": dob,
        "country": country,
        "email": email,
        "phone": phone,
        "address": address,
        "account_number": account_number,
        "account_balance": account_balance,
        "account_opened": account_opened,
        "monthly_transactions": monthly_transactions,
        "avg_transaction_amount": avg_transaction,
        "psi_key": psi_key,
        "psi_hash": psi_hash,
        "is_match": is_match
    }


def generate_server_dataset(num_records=5000, output_format="json"):
    """Generate server-side sanctions dataset"""
    
    print(f"Generating {num_records} sanctioned entities...")
    
    # Risk distribution: 10% CRITICAL, 30% HIGH, 40% MEDIUM, 20% LOW
    risk_distribution = [0.10, 0.30, 0.40, 0.20]
    
    entities = []
    for i in range(num_records):
        entity = generate_sanctioned_entity(i + 1, risk_distribution)
        entities.append(entity)
        
        if (i + 1) % 1000 == 0:
            print(f"  Generated {i + 1}/{num_records} entities...")
    
    # Save as JSON
    json_path = "data/server_data.json"
    with open(json_path, 'w') as f:
        json.dump(entities, f, indent=2)
    print(f"✓ Saved JSON: {json_path}")
    
    # Save as CSV
    csv_path = "data/server_data.csv"
    with open(csv_path, 'w', newline='') as f:
        if entities:
            writer = csv.DictWriter(f, fieldnames=entities[0].keys())
            writer.writeheader()
            writer.writerows(entities)
    print(f"✓ Saved CSV: {csv_path}")
    
    # Save PSI-ready format (just hashes)
    psi_path = "data/server_psi_hashes.txt"
    with open(psi_path, 'w') as f:
        for entity in entities:
            f.write(entity["psi_hash"] + "\n")
    print(f"✓ Saved PSI hashes: {psi_path}")
    
    # Statistics
    print("\n=== Server Dataset Statistics ===")
    print(f"Total entities: {len(entities)}")
    
    risk_counts = {}
    for entity in entities:
        risk = entity["risk_level"]
        risk_counts[risk] = risk_counts.get(risk, 0) + 1
    
    for risk, count in sorted(risk_counts.items()):
        print(f"  {risk}: {count} ({count/len(entities)*100:.1f}%)")
    
    program_counts = {}
    for entity in entities:
        prog = entity["sanction_program"]
        program_counts[prog] = program_counts.get(prog, 0) + 1
    
    print("\nTop sanction programs:")
    for prog, count in sorted(program_counts.items(), key=lambda x: x[1], reverse=True)[:5]:
        print(f"  {prog}: {count}")
    
    return entities


def generate_client_dataset(num_records=5000, match_percentage=2.0, server_entities=None):
    """Generate client-side customer dataset with controlled matches"""
    
    print(f"\nGenerating {num_records} customer records...")
    print(f"Target match rate: {match_percentage}%")
    
    num_matches = int(num_records * (match_percentage / 100))
    print(f"Creating {num_matches} intentional matches...")
    
    customers = []
    
    # Generate matching customers first
    if server_entities and num_matches > 0:
        # Randomly select entities to match
        matched_entities = random.sample(server_entities, min(num_matches, len(server_entities)))
        
        for i, entity in enumerate(matched_entities):
            customer = generate_customer(i + 1, is_match=True, sanctioned_entity=entity)
            customers.append(customer)
            
            if (i + 1) % 100 == 0:
                print(f"  Generated {i + 1}/{num_matches} matching customers...")
    
    # Generate clean customers
    num_clean = num_records - len(customers)
    print(f"Generating {num_clean} clean customers...")
    
    for i in range(num_clean):
        customer = generate_customer(len(customers) + 1, is_match=False)
        customers.append(customer)
        
        if (len(customers)) % 1000 == 0:
            print(f"  Generated {len(customers)}/{num_records} customers...")
    
    # Shuffle to mix matches and clean records
    random.shuffle(customers)
    
    # Update customer IDs to be sequential after shuffle
    for i, customer in enumerate(customers):
        customer["customer_id"] = f"CUST_{i + 1:06d}"
    
    # Save as JSON
    json_path = "data/client_data.json"
    with open(json_path, 'w') as f:
        json.dump(customers, f, indent=2)
    print(f"✓ Saved JSON: {json_path}")
    
    # Save as CSV
    csv_path = "data/client_data.csv"
    with open(csv_path, 'w', newline='') as f:
        if customers:
            writer = csv.DictWriter(f, fieldnames=customers[0].keys())
            writer.writeheader()
            writer.writerows(customers)
    print(f"✓ Saved CSV: {csv_path}")
    
    # Save PSI-ready format (just hashes)
    psi_path = "data/client_psi_hashes.txt"
    with open(psi_path, 'w') as f:
        for customer in customers:
            f.write(customer["psi_hash"] + "\n")
    print(f"✓ Saved PSI hashes: {psi_path}")
    
    # Save ground truth (matches) for validation
    matches = [c for c in customers if c["is_match"]]
    truth_path = "data/ground_truth_matches.json"
    with open(truth_path, 'w') as f:
        json.dump(matches, f, indent=2)
    print(f"✓ Saved ground truth: {truth_path}")
    
    # Statistics
    print("\n=== Client Dataset Statistics ===")
    print(f"Total customers: {len(customers)}")
    print(f"Actual matches: {len(matches)} ({len(matches)/len(customers)*100:.2f}%)")
    
    country_counts = {}
    for customer in customers:
        country = customer["country"]
        country_counts[country] = country_counts.get(country, 0) + 1
    
    print("\nTop countries:")
    for country, count in sorted(country_counts.items(), key=lambda x: x[1], reverse=True)[:10]:
        print(f"  {country}: {count}")
    
    total_balance = sum(c["account_balance"] for c in customers)
    avg_balance = total_balance / len(customers)
    print(f"\nAverage account balance: ${avg_balance:,.2f}")
    print(f"Total assets under management: ${total_balance:,.2f}")
    
    return customers


def main():
    parser = argparse.ArgumentParser(description="Generate synthetic datasets for PSI-based sanctions screening simulation")
    parser.add_argument("--server-size", type=int, default=5000, help="Number of sanctioned entities (default: 5000, max recommended for LE-PSI)")
    parser.add_argument("--client-size", type=int, default=5000, help="Number of customers (default: 5000)")
    parser.add_argument("--match-rate", type=float, default=3.0, help="Percentage of matches - realistic range 2-5%% (default: 3.0)")
    parser.add_argument("--format", choices=["json", "csv", "both"], default="both", help="Output format")
    
    args = parser.parse_args()
    
    print("=" * 70)
    print("    SYNTHETIC DATASET GENERATOR FOR FINANCIAL FRAUD DETECTION")
    print("         PSI-Based Sanctions Screening Simulation")
    print("=" * 70)
    print(f"Configuration:")
    print(f"  Server (Sanctions List): {args.server_size:,} entities")
    print(f"  Client (Customer Database): {args.client_size:,} records")
    print(f"  Match Rate (Hit Rate): {args.match_rate}%")
    print(f"  Expected Matches: {int(args.client_size * args.match_rate / 100):,}")
    print("=" * 70)
    
    # Generate server dataset (sanctions list)
    server_entities = generate_server_dataset(args.server_size, args.format)
    
    # Generate client dataset (customers)
    customers = generate_client_dataset(args.client_size, args.match_rate, server_entities)
    
    # Calculate actual statistics
    actual_matches = len([c for c in customers if c["is_match"]])
    actual_rate = (actual_matches / len(customers) * 100) if customers else 0
    
    print("\n" + "=" * 70)
    print("✓ DATASET GENERATION COMPLETE - READY FOR DEMONSTRATION")
    print("=" * 70)
    print("\nGenerated Files:")
    print("\n  📁 Server Dataset (Sanctions List):")
    print("     ├─ data/server_data.json       (Full entity details)")
    print("     ├─ data/server_data.csv        (Spreadsheet format)")
    print("     └─ data/server_psi_hashes.txt  (PSI input - hashes only)")
    print("\n  📁 Client Dataset (Customer Database):")
    print("     ├─ data/client_data.json       (Full customer details)")
    print("     ├─ data/client_data.csv        (Spreadsheet format)")
    print("     ├─ data/client_psi_hashes.txt  (PSI input - hashes only)")
    print("     └─ data/ground_truth_matches.json (Validation data)")
    print("\n" + "=" * 70)
    print("Research Presentation Statistics:")
    print(f"  • Total sanctioned entities: {len(server_entities):,}")
    print(f"  • Total customers screened: {len(customers):,}")
    print(f"  • Known matches (hits): {actual_matches:,} ({actual_rate:.2f}%)")
    print(f"  • Clean records: {len(customers) - actual_matches:,}")
    print("=" * 70)
    print("\nSimulation Workflow:")
    print("  1. Server: ./server_sim ../../data/server_psi_hashes.txt")
    print("  2. Client: ./client_sim ../../data/client_psi_hashes.txt")
    print("  3. Validate: Compare PSI results with ground_truth_matches.json")
    print("\nFor Research Presentation:")
    print("  • Show privacy: Server never sees customer names/data")
    print("  • Demonstrate accuracy: Validate against ground truth")
    print("  • Benchmark performance: Measure time for PSI computation")
    print("=" * 70)


if __name__ == "__main__":
    main()
