export function Skeleton({ className = '' }: { className?: string }) {
  return (
    <div className={`animate-pulse bg-paper/50 rounded ${className}`} />
  );
}

export function CardSkeleton() {
  return (
    <div className="animate-pulse bg-warm-white-light border border-paper/40 rounded-2xl p-5 sm:p-6">
      <div className="flex items-start gap-2 mb-2">
        <div className="flex-1">
          <div className="h-6 bg-paper/50 rounded w-1/3 mb-2" />
          <div className="h-4 bg-paper/50 rounded w-1/4 mb-2" />
          <div className="flex gap-2 mt-2">
            <div className="h-5 bg-paper/50 rounded w-12" />
            <div className="h-5 bg-paper/50 rounded w-16" />
          </div>
        </div>
      </div>
    </div>
  );
}

export function NameCardSkeleton() {
  return (
    <div className="animate-pulse bg-warm-white-light border border-paper/40 rounded-2xl p-5 sm:p-6">
      <div className="flex items-start justify-between gap-2 mb-3">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <div className="h-6 w-6 bg-paper/50 rounded" />
            <div className="h-7 bg-paper/50 rounded w-1/3" />
          </div>
          <div className="h-4 bg-paper/50 rounded w-1/5 mt-1.5 ml-8" />
        </div>
        <div className="flex gap-1">
          <div className="h-8 w-8 bg-paper/50 rounded-full" />
          <div className="h-8 w-8 bg-paper/50 rounded-full" />
        </div>
      </div>
      <div className="flex gap-2 ml-8 mb-3">
        <div className="h-5 bg-paper/50 rounded w-10" />
        <div className="h-5 bg-paper/50 rounded w-14" />
        <div className="h-5 bg-paper/50 rounded w-12" />
      </div>
      <div className="flex flex-wrap gap-x-3 gap-y-1 ml-8">
        {[1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="flex items-center gap-1.5">
            <div className="h-3 bg-paper/50 rounded w-12" />
            <div className="h-1.5 bg-paper/50 rounded w-12" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function ResultPageSkeleton() {
  return (
    <div className="max-w-4xl mx-auto p-4 space-y-4">
      {/* Navigation skeleton */}
      <div className="animate-pulse flex justify-between items-center mb-4">
        <div className="h-5 bg-paper/50 rounded w-20" />
        <div className="flex gap-2">
          <div className="h-8 w-8 bg-paper/50 rounded" />
          <div className="h-8 w-8 bg-paper/50 rounded" />
        </div>
      </div>
      {/* Bazi analysis skeleton */}
      <div className="animate-pulse bg-warm-white-light border border-paper/40 rounded-2xl p-5 space-y-3">
        <div className="h-6 bg-paper/50 rounded w-1/4" />
        <div className="h-4 bg-paper/50 rounded w-2/3" />
        <div className="grid grid-cols-5 gap-2">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="h-12 bg-paper/50 rounded" />
          ))}
        </div>
      </div>
      {/* Name cards grid skeleton */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4 mt-6">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <NameCardSkeleton key={i} />
        ))}
      </div>
    </div>
  );
}

export function FavoritesListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="animate-pulse space-y-3">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="bg-warm-white-light border border-paper/40 rounded-2xl p-5 flex items-center justify-between">
          <div className="space-y-2 flex-1">
            <div className="h-5 bg-paper/50 rounded w-1/4" />
            <div className="h-4 bg-paper/50 rounded w-1/3" />
            <div className="flex gap-2">
              <div className="h-4 bg-paper/50 rounded w-12" />
              <div className="h-4 bg-paper/50 rounded w-10" />
            </div>
          </div>
          <div className="h-8 w-8 bg-paper/50 rounded" />
        </div>
      ))}
    </div>
  );
}

export function HistoryListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="animate-pulse space-y-3">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="bg-warm-white-light border border-paper/40 rounded-2xl p-5 space-y-2">
          <div className="h-5 bg-paper/50 rounded w-1/3" />
          <div className="h-4 bg-paper/50 rounded w-1/2" />
          <div className="h-4 bg-paper/50 rounded w-1/4" />
        </div>
      ))}
    </div>
  );
}