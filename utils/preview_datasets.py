#!/usr/bin/env python3
"""
Preview generated datasets for research presentation
Shows sample records and statistics
"""

import json
import sys

def preview_server_data():
    """Show sample sanctioned entities"""
    print("\n" + "="*70)
    print("📊 SERVER DATASET (Sanctions List) - Sample Records")
    print("="*70)
    
    with open('data/server_data.json', 'r') as f:
        entities = json.load(f)
    
    # Show first 3 entities
    for i, entity in enumerate(entities[:3], 1):
        print(f"\n🔴 Sanctioned Entity #{i}:")
        print(f"   Entity ID: {entity['entity_id']}")
        print(f"   Name: {entity['name']}")
        if entity['aliases']:
            print(f"   Aliases: {', '.join(entity['aliases'])}")
        print(f"   DOB: {entity['dob']}")
        print(f"   Country: {entity['country']}")
        print(f"   Risk Level: {entity['risk_level']}")
        print(f"   Sanction Program: {entity['sanction_program']}")
        print(f"   Passport: {entity['passport_number']}")
        print(f"   PSI Key: {entity['psi_key']}")
        print(f"   PSI Hash: {entity['psi_hash'][:32]}...")
    
    print(f"\n... and {len(entities) - 3:,} more entities")


def preview_client_data():
    """Show sample customers including matches"""
    print("\n" + "="*70)
    print("📊 CLIENT DATASET (Customer Database) - Sample Records")
    print("="*70)
    
    with open('data/client_data.json', 'r') as f:
        customers = json.load(f)
    
    # Find and show matching customers
    matches = [c for c in customers if c['is_match']]
    clean = [c for c in customers if not c['is_match']]
    
    print(f"\n✅ Sample CLEAN Customer:")
    customer = clean[0]
    print(f"   Customer ID: {customer['customer_id']}")
    print(f"   Name: {customer['name']}")
    print(f"   DOB: {customer['dob']}")
    print(f"   Country: {customer['country']}")
    print(f"   Email: {customer['email']}")
    print(f"   Account: {customer['account_number']}")
    print(f"   Balance: ${customer['account_balance']:,.2f}")
    print(f"   Status: ✅ CLEAN (no sanction match)")
    
    print(f"\n🚨 Sample MATCHING Customer (Sanctions Hit):")
    customer = matches[0]
    print(f"   Customer ID: {customer['customer_id']}")
    print(f"   Name: {customer['name']}")
    print(f"   DOB: {customer['dob']}")
    print(f"   Country: {customer['country']}")
    print(f"   Email: {customer['email']}")
    print(f"   Account: {customer['account_number']}")
    print(f"   Balance: ${customer['account_balance']:,.2f}")
    print(f"   Status: 🚨 MATCH (appears in sanctions list)")
    print(f"   PSI Key: {customer['psi_key']}")
    
    print(f"\n📈 Dataset Summary:")
    print(f"   Total Customers: {len(customers):,}")
    print(f"   Clean Customers: {len(clean):,} ({len(clean)/len(customers)*100:.1f}%)")
    print(f"   Matches (Hits): {len(matches):,} ({len(matches)/len(customers)*100:.1f}%)")


def show_ground_truth():
    """Show ground truth matches for validation"""
    print("\n" + "="*70)
    print("🎯 GROUND TRUTH - Known Matches for PSI Validation")
    print("="*70)
    
    with open('data/ground_truth_matches.json', 'r') as f:
        matches = json.load(f)
    
    print(f"\nTotal Known Matches: {len(matches)}")
    print("\nFirst 5 matches:")
    
    for i, match in enumerate(matches[:5], 1):
        print(f"\n  Match #{i}:")
        print(f"    Customer ID: {match['customer_id']}")
        print(f"    Name: {match['name']}")
        print(f"    PSI Hash: {match['psi_hash'][:32]}...")
    
    print(f"\n... and {len(matches) - 5} more matches")
    print("\n💡 Use this file to validate your PSI results:")
    print("   - PSI should detect all these hashes")
    print("   - Accuracy = (PSI_matches / Ground_truth_matches) × 100%")


def show_research_metrics():
    """Display key metrics for research presentation"""
    print("\n" + "="*70)
    print("📊 RESEARCH PRESENTATION METRICS")
    print("="*70)
    
    with open('data/server_data.json', 'r') as f:
        server = json.load(f)
    
    with open('data/client_data.json', 'r') as f:
        client = json.load(f)
    
    matches = [c for c in client if c['is_match']]
    
    # Risk distribution
    risk_dist = {}
    for entity in server:
        risk = entity['risk_level']
        risk_dist[risk] = risk_dist.get(risk, 0) + 1
    
    # Country distribution for matches
    match_countries = {}
    for match in matches:
        country = match['country']
        match_countries[country] = match_countries.get(country, 0) + 1
    
    print(f"\n📈 Key Statistics:")
    print(f"   Sanctions Database Size: {len(server):,} entities")
    print(f"   Customer Database Size: {len(client):,} records")
    print(f"   Screening Hit Rate: {len(matches)/len(client)*100:.2f}%")
    print(f"   Total Expected PSI Comparisons: {len(server) * len(client):,}")
    
    print(f"\n🎯 Sanctions List Risk Profile:")
    for risk in ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW']:
        count = risk_dist.get(risk, 0)
        print(f"   {risk:8}: {count:4} ({count/len(server)*100:5.1f}%)")
    
    print(f"\n🌍 Top Countries in Matches:")
    top_countries = sorted(match_countries.items(), key=lambda x: x[1], reverse=True)[:5]
    for country, count in top_countries:
        print(f"   {country}: {count} matches")
    
    total_balance = sum(c['account_balance'] for c in client)
    match_balance = sum(c['account_balance'] for c in matches)
    
    print(f"\n💰 Financial Impact:")
    print(f"   Total AUM: ${total_balance:,.2f}")
    print(f"   Flagged Accounts: ${match_balance:,.2f}")
    print(f"   At-Risk Percentage: {match_balance/total_balance*100:.2f}%")


def main():
    print("\n" + "="*70)
    print("     DATASET PREVIEW - Financial Fraud Detection Research")
    print("="*70)
    
    try:
        preview_server_data()
        preview_client_data()
        show_ground_truth()
        show_research_metrics()
        
        print("\n" + "="*70)
        print("✓ Preview Complete - Datasets Ready for Simulation")
        print("="*70)
        print("\n💡 Next Steps:")
        print("   1. Review CSV files in Excel/Google Sheets")
        print("   2. Run PSI simulation with hash files")
        print("   3. Validate PSI results against ground_truth_matches.json")
        print("   4. Benchmark performance for research presentation")
        print("="*70 + "\n")
        
    except FileNotFoundError as e:
        print(f"\n❌ Error: Dataset files not found!")
        print(f"   Please run: python3 utils/generate_synthetic_datasets.py")
        sys.exit(1)


if __name__ == "__main__":
    main()
