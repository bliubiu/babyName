import Navigation from '@/components/Navigation';
import HomeContent from '@/components/HomeContent';

export default function Home() {
  return (
    <main className="min-h-screen py-6 md:py-12 px-4 md:px-6 cloud-bg">
      <div className="max-w-2xl mx-auto relative z-10">
        <Navigation showBackButton={false} showHistory={true} showFavorites={true} />
        <HomeContent />
      </div>
    </main>
  );
}