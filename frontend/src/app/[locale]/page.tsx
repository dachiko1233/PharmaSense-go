import Hero from "@/components/landing/Hero";
import ProblemSolution from "@/components/landing/ProblemSolution";
import Features from "@/components/landing/Features";
import DashboardPreview from "@/components/landing/DashboardPreview";
import Pricing from "@/components/landing/Pricing";
import CtaSection from "@/components/landing/CtaSection";
import Footer from "@/components/layout/Footer";

export default function RootPage() {
  return (
    <div className="min-h-screen bg-white">
      <Hero />
      <ProblemSolution />
      <Features />
      <DashboardPreview />
      <Pricing />
      <CtaSection />
      <Footer />
    </div>
  );
}
