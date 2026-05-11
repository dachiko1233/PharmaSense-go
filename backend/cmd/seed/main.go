package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"pharmasense/internal/config"
	"pharmasense/internal/db"
	"pharmasense/internal/domain"
	"pharmasense/internal/services"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg := config.Load()
	ctx := context.Background()

	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("seeding database...")
	if err := seed(ctx, pool); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
	slog.Info("seed complete")
}

func seed(ctx context.Context, pool *pgxpool.Pool) error {
	// Wipe existing seeded data
	tables := []string{
		"alert_actions", "risk_assessments", "sales", "inventory_batches",
		"pharmacy_users", "users", "pharmacies", "chains", "products",
	}
	for _, t := range tables {
		if _, err := pool.Exec(ctx, fmt.Sprintf("DELETE FROM %s", t)); err != nil {
			return fmt.Errorf("wipe %s: %w", t, err)
		}
	}
	slog.Info("tables wiped")

	// Create chain
	var chainID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO chains (name, owner_email) VALUES ($1, $2) RETURNING id
	`, "Nicosia Health Group", "chain_admin@pharmasense.cy").Scan(&chainID)
	if err != nil {
		return fmt.Errorf("create chain: %w", err)
	}

	// Create pharmacies
	type PharmDef struct {
		Name    string
		License string
		City    string
		Plan    string
	}
	pharmDefs := []PharmDef{
		{"Nicosia Central Pharmacy", "CY-PH-2024-001", "Nicosia", "pro"},
		{"Limassol Marina Pharmacy", "CY-PH-2024-002", "Limassol", "free"},
		{"Paphos Tourist Pharmacy", "CY-PH-2024-003", "Paphos", "chain"},
	}
	pharmacyIDs := make([]uuid.UUID, 3)
	for i, def := range pharmDefs {
		var id uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO pharmacies (chain_id, name, license_number, city, plan, stripe_customer_id, stripe_subscription_id, subscription_status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
		`, chainID, def.Name, def.License, def.City, def.Plan,
			fmt.Sprintf("cus_mock_%d", i+1),
			fmt.Sprintf("sub_mock_%d", i+1),
			"active",
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("create pharmacy %s: %w", def.Name, err)
		}
		pharmacyIDs[i] = id
	}
	slog.Info("pharmacies created", "count", 3)

	// Hash passwords (bcrypt cost 12)
	hash, err := bcrypt.GenerateFromPassword([]byte("Demo1234!"), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	passwordHash := string(hash)

	// Create users
	type UserDef struct {
		Email    string
		FullName string
	}
	userDefs := []UserDef{
		{"chain_admin@pharmasense.cy", "Alexandra Papadopoulos"},
		{"admin@pharmasense.cy", "Nikos Stavrou"},
		{"staff@pharmasense.cy", "Maria Christodoulou"},
	}
	userIDs := make([]uuid.UUID, 3)
	for i, def := range userDefs {
		var id uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO users (default_pharmacy_id, email, password_hash, full_name, email_verified, is_active)
			VALUES ($1, $2, $3, $4, true, true)
			RETURNING id
		`, pharmacyIDs[0], def.Email, passwordHash, def.FullName).Scan(&id)
		if err != nil {
			return fmt.Errorf("create user %s: %w", def.Email, err)
		}
		userIDs[i] = id
	}
	slog.Info("users created", "count", 3)

	// Link users to pharmacies
	// chain_admin → all 3 as chain_admin
	for _, pid := range pharmacyIDs {
		_, err := pool.Exec(ctx, `
			INSERT INTO pharmacy_users (pharmacy_id, user_id, role) VALUES ($1, $2, 'chain_admin')
		`, pid, userIDs[0])
		if err != nil {
			return fmt.Errorf("link chain_admin: %w", err)
		}
	}
	// admin → pharmacy 0 as admin
	if _, err := pool.Exec(ctx, `
		INSERT INTO pharmacy_users (pharmacy_id, user_id, role) VALUES ($1, $2, 'admin')
	`, pharmacyIDs[0], userIDs[1]); err != nil {
		return fmt.Errorf("link admin: %w", err)
	}
	// staff → pharmacy 0 as staff
	if _, err := pool.Exec(ctx, `
		INSERT INTO pharmacy_users (pharmacy_id, user_id, role) VALUES ($1, $2, 'staff')
	`, pharmacyIDs[0], userIDs[2]); err != nil {
		return fmt.Errorf("link staff: %w", err)
	}

	// Create products
	productIDs, err := createProducts(ctx, pool)
	if err != nil {
		return fmt.Errorf("create products: %w", err)
	}
	slog.Info("products created", "count", len(productIDs))

	// Create batches and sales for each pharmacy
	rng := rand.New(rand.NewSource(42))
	for i, pid := range pharmacyIDs {
		slog.Info("seeding pharmacy", "index", i+1, "pharmacy_id", pid)
		if err := seedPharmacyData(ctx, pool, pid, productIDs, rng); err != nil {
			return fmt.Errorf("seed pharmacy %d: %w", i, err)
		}
	}

	slog.Info("running risk calculations...")
	for _, pid := range pharmacyIDs {
		if err := runRiskEngine(ctx, pool, pid, rng); err != nil {
			slog.Warn("risk engine failed", "pharmacy", pid, "error", err)
		}
	}

	return nil
}

type ProductDef struct {
	Name     string
	NameEl   string
	Category string
	Manuf    string
	RxReq    bool
}

func productCatalog() []ProductDef {
	return []ProductDef{
		// Painkillers
		{"Paracetamol 500mg Tablets", "Παρακεταμόλη 500mg", "Painkillers", "GSK", false},
		{"Ibuprofen 400mg Capsules", "Ιβουπροφαίνη 400mg", "Painkillers", "Bayer", false},
		{"Aspirin 100mg Tablets", "Ασπιρίνη 100mg", "Painkillers", "Bayer", false},
		{"Diclofenac Gel 1%", "Δικλοφενάκ Gel 1%", "Painkillers", "Novartis", false},
		{"Naproxen 250mg Tablets", "Ναπροξένη 250mg", "Painkillers", "Pfizer", false},
		{"Codeine Phosphate 30mg", "Κωδεΐνη Φωσφορική 30mg", "Painkillers", "Sanofi", true},
		{"Tramadol 50mg Capsules", "Τραμαδόλη 50mg", "Painkillers", "Grünenthal", true},

		// Antibiotics
		{"Amoxicillin 500mg Capsules", "Αμοξικιλλίνη 500mg", "Antibiotics", "Pfizer", true},
		{"Azithromycin 250mg Tablets", "Αζιθρομυκίνη 250mg", "Antibiotics", "Pfizer", true},
		{"Clarithromycin 500mg Tablets", "Κλαριθρομυκίνη 500mg", "Antibiotics", "Abbott", true},
		{"Ciprofloxacin 500mg Tablets", "Σιπροφλοξασίνη 500mg", "Antibiotics", "Bayer", true},
		{"Metronidazole 400mg Tablets", "Μετρονιδαζόλη 400mg", "Antibiotics", "Sanofi", true},
		{"Doxycycline 100mg Capsules", "Δοξυκυκλίνη 100mg", "Antibiotics", "Pfizer", true},
		{"Penicillin V 500mg Tablets", "Πενικιλλίνη V 500mg", "Antibiotics", "GSK", true},

		// Vitamins & Supplements
		{"Vitamin C 1000mg Effervescent", "Βιταμίνη C 1000mg Αναβράζον", "Vitamins", "Bayer", false},
		{"Vitamin D3 2000 IU Drops", "Βιταμίνη D3 2000 IU Σταγόνες", "Vitamins", "Roche", false},
		{"Vitamin B Complex Tablets", "Βιταμίνη Β Σύμπλεγμα", "Vitamins", "Pfizer", false},
		{"Omega-3 Fish Oil 1000mg", "Ωμέγα-3 Ιχθυέλαιο 1000mg", "Vitamins", "Sanofi", false},
		{"Magnesium 375mg Tablets", "Μαγνήσιο 375mg", "Vitamins", "Roche", false},
		{"Zinc 15mg Tablets", "Ψευδάργυρος 15mg", "Vitamins", "Bayer", false},
		{"Iron 65mg Tablets", "Σίδηρος 65mg", "Vitamins", "Sanofi", false},
		{"Folic Acid 5mg Tablets", "Φυλλικό Οξύ 5mg", "Vitamins", "GSK", false},
		{"Calcium 600mg + D3", "Ασβέστιο 600mg + D3", "Vitamins", "Pfizer", false},
		{"Multivitamin Daily Tablets", "Πολυβιταμίνες Ημέρας", "Vitamins", "Roche", false},
		{"Vitamin E 400 IU Capsules", "Βιταμίνη E 400 IU", "Vitamins", "Bayer", false},

		// Cold & Flu
		{"Fludrex Syrup 150ml", "Φλούντρεξ Σιρόπι 150ml", "Cold/Flu", "Sanofi", false},
		{"Rhinathiol Expectorant 200ml", "Ρινατιόλη Εκκριτική 200ml", "Cold/Flu", "Sanofi", false},
		{"Strepsils Throat Lozenges", "Στρέψιλς Παστίλιες Λαιμού", "Cold/Flu", "Reckitt", false},
		{"Sinusitis Nasal Spray 10ml", "Σπρέι Ρινός Για Ιγμορίτιδα", "Cold/Flu", "GSK", false},
		{"Vicks VapoRub 50g", "Vicks VapoRub 50g", "Cold/Flu", "Procter & Gamble", false},
		{"Lemsip Max Cold Sachets", "Lemsip Μέγιστη Δόση Σακουλάκια", "Cold/Flu", "Reckitt", false},
		{"Otrivin Nasal Spray 10ml", "Otrivin Ρινικό Σπρέι 10ml", "Cold/Flu", "Novartis", false},

		// Allergy
		{"Cetirizine 10mg Tablets", "Σετιριζίνη 10mg", "Allergy", "GSK", false},
		{"Loratadine 10mg Tablets", "Λοραταδίνη 10mg", "Allergy", "Schering", false},
		{"Fexofenadine 120mg Tablets", "Φεξοφεναδίνη 120mg", "Allergy", "Sanofi", false},
		{"Dexamethasone Eye Drops 5ml", "Δεξαμεθαζόνη Οφθαλμικές Σταγόνες", "Allergy", "Pfizer", true},
		{"Mometasone Nasal Spray", "Μομεταζόνη Ρινικό Σπρέι", "Allergy", "Schering", false},
		{"Hydrocortisone Cream 1% 30g", "Υδροκορτιζόνη Κρέμα 1%", "Allergy", "Bayer", false},

		// Digestive
		{"Omeprazole 20mg Capsules", "Ομεπραζόλη 20mg", "Digestive", "AstraZeneca", false},
		{"Pantoprazole 40mg Tablets", "Παντοπραζόλη 40mg", "Digestive", "Pfizer", true},
		{"Loperamide 2mg Capsules", "Λοπεραμίδη 2mg", "Digestive", "Janssen", false},
		{"Metoclopramide 10mg Tablets", "Μετοκλοπραμίδη 10mg", "Digestive", "Sanofi", false},
		{"Domperidone 10mg Tablets", "Δομπεριδόνη 10mg", "Digestive", "Janssen", false},
		{"Lactulose Solution 300ml", "Λακτουλόζη Διάλυμα 300ml", "Digestive", "Sanofi", false},
		{"Gaviscon Advance Liquid 300ml", "Gaviscon Advance Υγρό 300ml", "Digestive", "Reckitt", false},
		{"Buscopan 10mg Tablets", "Buscopan 10mg", "Digestive", "Bayer", false},

		// Cardiovascular
		{"Atorvastatin 20mg Tablets", "Ατορβαστατίνη 20mg", "Cardiovascular", "Pfizer", true},
		{"Amlodipine 5mg Tablets", "Αμλοδιπίνη 5mg", "Cardiovascular", "Pfizer", true},
		{"Ramipril 5mg Tablets", "Ραμιπρίλη 5mg", "Cardiovascular", "Sanofi", true},
		{"Bisoprolol 5mg Tablets", "Βισοπρολόλη 5mg", "Cardiovascular", "Merck", true},
		{"Clopidogrel 75mg Tablets", "Κλοπιδογρέλη 75mg", "Cardiovascular", "Sanofi", true},
		{"Warfarin 5mg Tablets", "Βαρφαρίνη 5mg", "Cardiovascular", "Teva", true},
		{"Furosemide 40mg Tablets", "Φουροσεμίδη 40mg", "Cardiovascular", "Sanofi", true},

		// Diabetes
		{"Metformin 500mg Tablets", "Μετφορμίνη 500mg", "Diabetes", "Merck", true},
		{"Metformin 850mg Tablets", "Μετφορμίνη 850mg", "Diabetes", "Merck", true},
		{"Glibenclamide 5mg Tablets", "Γλιβενκλαμίδη 5mg", "Diabetes", "Sanofi", true},
		{"Sitagliptin 100mg Tablets", "Σιταγλιπτίνη 100mg", "Diabetes", "MSD", true},
		{"Insulin Glargine 300 Units/ml", "Ινσουλίνη Γλαργκίνη 300U/ml", "Diabetes", "Sanofi", true},
		{"Glucose Test Strips (50 pack)", "Ταινίες Γλυκόζης (50 τεμ)", "Diabetes", "Roche", false},
		{"Glucometer Starter Kit", "Starter Kit Γλυκόμετρου", "Diabetes", "Roche", false},

		// Skincare
		{"Eucerin pH5 Lotion 400ml", "Eucerin pH5 Λοσιόν 400ml", "Skincare", "Beiersdorf", false},
		{"Bepanthen Cream 30g", "Bepanthen Κρέμα 30g", "Skincare", "Bayer", false},
		{"Cetaphil Moisturising Cream 250g", "Cetaphil Ενυδατική Κρέμα 250g", "Skincare", "Galderma", false},
		{"Sudocrem Antiseptic Healing Cream", "Sudocrem Αντισηπτική Κρέμα", "Skincare", "Forest Laboratories", false},
		{"Nivea Sensitive Shave Gel 200ml", "Nivea Sensitive Gel Ξυρίσματος", "Skincare", "Beiersdorf", false},
		{"Clindamycin Phosphate Gel 1%", "Κλινδαμυκίνη Φωσφορική Gel 1%", "Skincare", "Pfizer", true},
		{"Tretinoin Cream 0.05%", "Τρετινοΐνη Κρέμα 0.05%", "Skincare", "Roche", true},
		{"Fluconazole 150mg Capsule", "Φλουκοναζόλη 150mg", "Skincare", "Pfizer", true},

		// Baby Care
		{"Calpol Infant Suspension 100ml", "Calpol Παιδικό Εναιώρημα 100ml", "Baby Care", "Johnson & Johnson", false},
		{"Nurofen for Children 100ml", "Nurofen Παιδικό 100ml", "Baby Care", "Reckitt", false},
		{"Infacol Colic Relief 50ml", "Infacol Κολικός 50ml", "Baby Care", "Forest Laboratories", false},
		{"Sudafed Baby Drops 15ml", "Σταγόνες Μύτης Μωρού 15ml", "Baby Care", "Pfizer", false},
		{"Dentinox Teething Gel 10g", "Dentinox Gel Οδοντοφυΐας", "Baby Care", "Dentinox", false},
		{"Johnsons Baby Lotion 300ml", "Johnsons Λοσιόν Μωρού 300ml", "Baby Care", "Johnson & Johnson", false},
		{"Pampers Sensitive Wipes 80pk", "Pampers Sensitive Μαντηλάκια", "Baby Care", "Procter & Gamble", false},

		// First Aid
		{"Elastoplast Assorted Plasters 40pk", "Elastoplast Αυτοκόλλητα 40τεμ", "First Aid", "Beiersdorf", false},
		{"Savlon Antiseptic Cream 100g", "Savlon Αντισηπτική Κρέμα 100g", "First Aid", "Novartis", false},
		{"Sterile Gauze Swabs 10x10cm", "Αποστειρωμένες Γάζες 10x10cm", "First Aid", "Hartmann", false},
		{"Triangular Bandage 96x96cm", "Τριγωνικός Επίδεσμος", "First Aid", "Hartmann", false},
		{"Digital Thermometer", "Ψηφιακό Θερμόμετρο", "First Aid", "Omron", false},
		{"Hydrogen Peroxide 3% 100ml", "Υπεροξείδιο Υδρογόνου 3%", "First Aid", "Bayer", false},
		{"Medical Alcohol 70% 100ml", "Ιατρική Αλκοόλη 70%", "First Aid", "Generic", false},

		// Additional products to reach ~150
		{"Salbutamol Inhaler 100mcg", "Σαλβουταμόλη Εισπνευστήρας", "Respiratory", "GSK", true},
		{"Beclomethasone Inhaler 250mcg", "Βεκλομεθαζόνη Εισπνευστήρας", "Respiratory", "Chiesi", true},
		{"Montelukast 10mg Tablets", "Μοντελουκάστ 10mg", "Respiratory", "MSD", true},
		{"Ipratropium Bromide 0.02% Nebules", "Ιπρατρόπιο Βρωμίδιο 0.02%", "Respiratory", "Boehringer", true},
		{"N-acetylcysteine 600mg Sachets", "N-ακετυλκυστεΐνη 600mg", "Respiratory", "Zambon", false},

		{"Sertraline 50mg Tablets", "Σερτραλίνη 50mg", "Mental Health", "Pfizer", true},
		{"Escitalopram 10mg Tablets", "Εσκιταλοπράμη 10mg", "Mental Health", "Lundbeck", true},
		{"Diazepam 5mg Tablets", "Διαζεπάμη 5mg", "Mental Health", "Roche", true},
		{"Melatonin 1mg Tablets", "Μελατονίνη 1mg", "Mental Health", "Teva", false},
		{"Valerian Root Extract 450mg", "Εκχύλισμα Ριζόχορτου 450mg", "Mental Health", "Various", false},

		{"Levothyroxine 50mcg Tablets", "Λεβοθυροξίνη 50mcg", "Thyroid", "Merck", true},
		{"Levothyroxine 100mcg Tablets", "Λεβοθυροξίνη 100mcg", "Thyroid", "Merck", true},
		{"Propylthiouracil 50mg Tablets", "Προπυλθειουρακίλη 50mg", "Thyroid", "Teva", true},

		{"Timolol Eye Drops 0.5%", "Τιμολόλη Οφθαλμικές Σταγόνες 0.5%", "Eye Care", "MSD", true},
		{"Latanoprost 0.005% Eye Drops", "Λαταναπροστ 0.005%", "Eye Care", "Pfizer", true},
		{"Sodium Hyaluronate Eye Drops", "Υαλουρονικό Νάτριο Οφθαλμικό", "Eye Care", "Thea", false},
		{"Chloramphenicol Eye Drops 0.5%", "Χλωραμφαινικόλη Οφθαλμικές 0.5%", "Eye Care", "Bausch", true},

		{"Fluoxetine 20mg Capsules", "Φλουοξετίνη 20mg", "Mental Health", "Lilly", true},
		{"Quetiapine 25mg Tablets", "Κουετιαπίνη 25mg", "Mental Health", "AstraZeneca", true},
		{"Gabapentin 300mg Capsules", "Γκαμπαπεντίνη 300mg", "Mental Health", "Pfizer", true},
		{"Pregabalin 75mg Capsules", "Πρεγκαμπαλίνη 75mg", "Mental Health", "Pfizer", true},

		{"Clotrimazole Cream 1% 30g", "Κλοτριμαζόλη Κρέμα 1%", "Antifungal", "Bayer", false},
		{"Miconazole Gel 2% Oral", "Μικοναζόλη Gel 2% Στοματικό", "Antifungal", "Janssen", false},
		{"Terbinafine 1% Cream 30g", "Τερμπιναφίνη 1% Κρέμα", "Antifungal", "Novartis", false},

		{"Heparin Sodium Gel 50000 IU", "Ηπαρίνη Νατρίου Gel", "Cardiovascular", "Roche", false},
		{"Glyceryl Trinitrate Spray 400mcg", "Νιτρογλυκερίνη Σπρέι 400mcg", "Cardiovascular", "Pfizer", true},

		{"Oral Rehydration Salts Sachets", "Άλατα Ενυδάτωσης Σακουλάκια", "Digestive", "Sanofi", false},
		{"Probiotic Capsules (30 capsules)", "Προβιοτικές Κάψουλες (30τεμ)", "Digestive", "Various", false},
		{"Senna 7.5mg Tablets", "Σέννα 7.5mg", "Digestive", "Norgine", false},

		{"Aciclovir 5% Cream 2g", "Ακυκλοβίρη 5% Κρέμα 2g", "Antiviral", "GSK", false},
		{"Valaciclovir 500mg Tablets", "Βαλακυκλοβίρη 500mg", "Antiviral", "GSK", true},

		{"Ketorolac Injection 30mg/ml", "Κετορολάκη Ένεση 30mg/ml", "Painkillers", "Roche", true},
		{"Morphine Sulfate 10mg/5ml Solution", "Θειική Μορφίνη 10mg/5ml", "Painkillers", "Sanofi", true},
	}
}

func createProducts(ctx context.Context, pool *pgxpool.Pool) ([]uuid.UUID, error) {
	catalog := productCatalog()
	ids := make([]uuid.UUID, 0, len(catalog))

	for _, prod := range catalog {
		var id uuid.UUID
		err := pool.QueryRow(ctx, `
			INSERT INTO products (name, name_el, category, manufacturer, requires_prescription)
			VALUES ($1, $2, $3, $4, $5) RETURNING id
		`, prod.Name, prod.NameEl, prod.Category, prod.Manuf, prod.RxReq).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("insert product %s: %w", prod.Name, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func seedPharmacyData(ctx context.Context, pool *pgxpool.Pool, pharmacyID uuid.UUID, productIDs []uuid.UUID, rng *rand.Rand) error {
	suppliers := []string{"MedSupply Cyprus", "PharmaWholesale Ltd", "EuroMeds"}
	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Risk distribution: 20% CRITICAL / 25% HIGH / 20% MEDIUM / 35% LOW
	type BatchSpec struct {
		daysExpiry  int
		qty         int
		avgSales    float64
		riskLevel   string
	}

	batchCount := 0
	for _, productID := range productIDs {
		// 3-4 batches per product per pharmacy
		numBatches := 3 + rng.Intn(2)
		for b := 0; b < numBatches; b++ {
			riskRoll := rng.Float64()
			var spec BatchSpec

			switch {
			case riskRoll < 0.20: // CRITICAL
				spec.daysExpiry = 5 + rng.Intn(25)     // 5-30 days
				spec.qty = 50 + rng.Intn(150)           // high qty
				spec.avgSales = float64(rng.Intn(3) + 1) // low sales
				spec.riskLevel = domain.RiskLevelCritical

			case riskRoll < 0.45: // HIGH
				spec.daysExpiry = 31 + rng.Intn(60)    // 31-90 days
				spec.qty = 100 + rng.Intn(200)
				spec.avgSales = float64(rng.Intn(2) + 1)
				spec.riskLevel = domain.RiskLevelHigh

			case riskRoll < 0.65: // MEDIUM
				spec.daysExpiry = 91 + rng.Intn(90)    // 91-180 days
				spec.qty = 60 + rng.Intn(100)
				spec.avgSales = float64(rng.Intn(3) + 1)
				spec.riskLevel = domain.RiskLevelMedium

			default: // LOW
				spec.daysExpiry = 181 + rng.Intn(365)  // 181-546 days
				spec.qty = 20 + rng.Intn(80)
				spec.avgSales = float64(rng.Intn(5) + 2)
				spec.riskLevel = domain.RiskLevelLow
			}

			expiryDate := today.AddDate(0, 0, spec.daysExpiry)
			receivedDate := today.AddDate(0, 0, -(30 + rng.Intn(60)))
			purchasePrice := 0.50 + rng.Float64()*49.50
			sellingPrice := purchasePrice * (1.2 + rng.Float64()*0.5)
			batchNum := fmt.Sprintf("BN-%d-%04d", today.Year(), rng.Intn(9999)+1)
			supplier := suppliers[rng.Intn(len(suppliers))]

			var batchID uuid.UUID
			err := pool.QueryRow(ctx, `
				INSERT INTO inventory_batches
				  (pharmacy_id, product_id, batch_number, expiry_date, initial_quantity, current_quantity,
				   purchase_price, selling_price, supplier, received_date)
				VALUES ($1,$2,$3,$4,$5,$5,$6,$7,$8,$9)
				RETURNING id
			`, pharmacyID, productID, batchNum, expiryDate, spec.qty,
				roundPrice(purchasePrice), roundPrice(sellingPrice), supplier, receivedDate,
			).Scan(&batchID)
			if err != nil {
				continue
			}
			batchCount++

			// Create sales for last 90 days
			if err := createSales(ctx, pool, pharmacyID, batchID, productID, spec.avgSales, spec.qty, today, rng); err != nil {
				continue
			}
		}
	}

	slog.Info("batches seeded", "pharmacy_id", pharmacyID, "count", batchCount)
	return nil
}

func createSales(ctx context.Context, pool *pgxpool.Pool, pharmacyID, batchID, productID uuid.UUID,
	avgDaily float64, maxQty int, today time.Time, rng *rand.Rand) error {

	for dayOffset := 90; dayOffset >= 1; dayOffset-- {
		saleDate := today.AddDate(0, 0, -dayOffset)

		// Poisson-like random daily sales
		dailyQty := 0
		for dailyQty < int(avgDaily*2) {
			if rng.Float64() < avgDaily/float64(int(avgDaily*2)+1) {
				dailyQty += rng.Intn(3) + 1
			}
			break
		}
		if dailyQty == 0 {
			continue
		}
		if dailyQty > maxQty/10 {
			dailyQty = maxQty / 10
		}
		if dailyQty < 1 {
			continue
		}

		unitPrice := 1.0 + rng.Float64()*50.0
		_, err := pool.Exec(ctx, `
			INSERT INTO sales (pharmacy_id, batch_id, product_id, quantity, unit_price, total_amount, sale_date)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, pharmacyID, batchID, productID, dailyQty, roundPrice(unitPrice),
			roundPrice(float64(dailyQty)*unitPrice), saleDate)
		if err != nil {
			return err
		}
	}
	return nil
}

func runRiskEngine(ctx context.Context, pool *pgxpool.Pool, pharmacyID uuid.UUID, rng *rand.Rand) error {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	rows, err := pool.Query(ctx, `
		SELECT ib.id, ib.expiry_date, ib.current_quantity, ib.purchase_price,
		       COALESCE(
		         (SELECT SUM(s.quantity)::float / 90.0
		          FROM sales s
		          WHERE s.batch_id = ib.id
		            AND s.sale_date >= $2 - INTERVAL '90 days'),
		         0.5
		       ) as avg_daily_sales
		FROM inventory_batches ib
		WHERE ib.pharmacy_id = $1
	`, pharmacyID, today)
	if err != nil {
		return err
	}
	defer rows.Close()

	type BatchData struct {
		ID            uuid.UUID
		ExpiryDate    time.Time
		CurrentQty    int
		PurchasePrice float64
		AvgDailySales float64
	}

	var batches []BatchData
	for rows.Next() {
		var bd BatchData
		if err := rows.Scan(&bd.ID, &bd.ExpiryDate, &bd.CurrentQty, &bd.PurchasePrice, &bd.AvgDailySales); err != nil {
			continue
		}
		batches = append(batches, bd)
	}
	rows.Close()

	for _, bd := range batches {
		result := services.CalculateRisk(services.RiskInput{
			CurrentQuantity: bd.CurrentQty,
			ExpiryDate:      bd.ExpiryDate,
			AvgDailySales:   bd.AvgDailySales,
			PurchasePrice:   bd.PurchasePrice,
			Today:           today,
		})

		expectedSales := result.ExpectedSales
		surplus := result.EstimatedSurplus
		loss := result.EstimatedLoss
		disc := result.SuggestedDiscountPct
		avgSales := bd.AvgDailySales

		_, err := pool.Exec(ctx, `
			INSERT INTO risk_assessments
			  (batch_id, pharmacy_id, risk_level, days_until_expiry, avg_daily_sales,
			   expected_sales, estimated_surplus, estimated_loss, suggested_discount_percent)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		`, bd.ID, pharmacyID, result.RiskLevel, result.DaysUntilExpiry, avgSales,
			expectedSales, surplus, loss, disc)
		if err != nil {
			slog.Warn("risk insert failed", "batch_id", bd.ID, "error", err)
		}
	}

	return nil
}

func roundPrice(v float64) float64 {
	return float64(int(v*100)) / 100
}
