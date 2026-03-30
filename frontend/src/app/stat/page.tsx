'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { getNameStats, getProvinceStats, NameStat } from '@/lib/api';
import { useToast } from '@/components/Toast';

const provinces = ['北京', '上海', '广东', '浙江', '江苏', '四川', '湖北', '湖南', '山东', '河南'];

export default function StatPage() {
  const router = useRouter();
  const { showToast } = useToast();
  const [searchName, setSearchName] = useState('');
  const [selectedProvince, setSelectedProvince] = useState('北京');
  const [stats, setStats] = useState<NameStat | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const handleSearch = async () => {
    if (!searchName.trim()) {
      showToast('请输入要查询的名字', 'warning');
      return;
    }

    setIsLoading(true);
    try {
      const res = await getNameStats(searchName.trim());
      if (res.success) {
        setStats(res.data);
      } else {
        showToast('查询失败', 'error');
      }
    } catch (error) {
      console.error('Error searching name:', error);
      showToast('查询失败', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleProvinceSearch = async () => {
    if (!searchName.trim()) {
      showToast('请输入要查询的名字', 'warning');
      return;
    }

    setIsLoading(true);
    try {
      const res = await getProvinceStats(searchName.trim(), selectedProvince);
      if (res.success) {
        setStats(res.data);
      } else {
        showToast('查询失败', 'error');
      }
    } catch (error) {
      console.error('Error searching province stats:', error);
      showToast('查询失败', 'error');
    } finally {
      setIsLoading(false);
    }
  };

  const getRateLevel = (rate: number) => {
    if (rate >= 5) return { text: '极高', color: 'text-red-500', description: '重名风险高，建议考虑其他名字' };
    if (rate >= 2) return { text: '较高', color: 'text-orange-500', description: '较常见，可以考虑其他名字' };
    if (rate >= 0.5) return { text: '一般', color: 'text-yellow-500', description: '正常水平，可以接受' };
    if (rate >= 0.001) return { text: '较低', color: 'text-green-500', description: '较少见，较为独特' };
    return { text: '极低', color: 'text-emerald-600', description: '稀有独特，非常少见' };
  };

  return (
    <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
      <div className="max-w-2xl mx-auto">
        <div className="flex justify-between items-center mb-4 md:mb-6">
          <button
            onClick={() => router.push('/')}
            className="flex items-center gap-2 text-teal hover:text-crimson transition-colors text-sm md:text-base"
          >
            ← 返回首页
          </button>
          <h1 className="text-xl md:text-2xl font-bold text-ink">重名率查询</h1>
          <div className="w-20"></div>
        </div>

        <div className="card">
          <div className="mb-4">
            <label className="block text-sm text-teal mb-2">输入名字（单字或双字）</label>
            <input
              type="text"
              value={searchName}
              onChange={(e) => setSearchName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              placeholder="例如：伟、芳、明、娜"
              className="w-full px-4 py-2 border border-warm-white rounded-lg focus:outline-none focus:ring-2 focus:ring-crimson text-ink bg-white"
              maxLength={2}
            />
          </div>

          <div className="flex gap-2 mb-4">
            <button
              onClick={handleSearch}
              disabled={isLoading}
              className="flex-1 px-4 py-2 bg-crimson text-white rounded-lg hover:bg-crimson/90 transition-colors disabled:opacity-50"
            >
              {isLoading ? '查询中...' : '全国查询'}
            </button>
          </div>

          <div className="border-t border-warm-white pt-4">
            <label className="block text-sm text-teal mb-2">省份查询</label>
            <div className="flex gap-2 mb-2">
              <select
                value={selectedProvince}
                onChange={(e) => setSelectedProvince(e.target.value)}
                className="flex-1 px-3 py-2 border border-warm-white rounded-lg focus:outline-none focus:ring-2 focus:ring-crimson text-ink bg-white"
              >
                {provinces.map((p) => (
                  <option key={p} value={p}>{p}</option>
                ))}
              </select>
              <button
                onClick={handleProvinceSearch}
                disabled={isLoading}
                className="px-4 py-2 bg-teal text-white rounded-lg hover:bg-teal/90 transition-colors disabled:opacity-50"
              >
                {isLoading ? '...' : '查询'}
              </button>
            </div>
          </div>

          {stats && (
            <div className="mt-6 p-4 bg-warm-white rounded-lg animate-fadeInUp">
              <h3 className="text-lg font-medium text-ink mb-4">查询结果</h3>
              
              <div className="grid grid-cols-2 gap-4">
                <div className="text-center p-3 bg-white rounded-lg">
                  <p className="text-2xl md:text-3xl font-bold text-crimson">{stats.count.toLocaleString()}</p>
                  <p className="text-sm text-teal">使用人数</p>
                </div>
                <div className="text-center p-3 bg-white rounded-lg">
                  <p className="text-2xl md:text-3xl font-bold text-gold">{stats.rate.toFixed(4)}%</p>
                  <p className="text-sm text-teal">重名率</p>
                </div>
              </div>

              <div className="mt-4 text-center">
                <p className="text-teal text-sm mb-2">
                  重名程度：<span className={`font-medium ${getRateLevel(stats.rate).color}`}>{getRateLevel(stats.rate).text}</span>
                </p>
                <p className="text-xs text-teal">
                  {getRateLevel(stats.rate).description}
                </p>
                {stats.province && (
                  <p className="text-teal text-sm mt-2">
                    数据来源：{stats.province}
                  </p>
                )}
              </div>

              <div className="mt-4 p-3 bg-white rounded-lg">
                <p className="text-sm text-teal">
                  {stats.rate > 0 ? (
                    <>在全国约 14 亿人口中，共有约 <span className="text-crimson font-medium">{stats.count.toLocaleString()}</span> 人使用此名字，重名率约为 <span className="text-gold font-medium">{stats.rate.toFixed(4)}%</span>。</>
                  ) : (
                    <>该名字在全国使用人数较少，属于稀有名字。</>
                  )}
                </p>
              </div>
            </div>
          )}

          <div className="mt-6 p-4 bg-warm-white rounded-lg">
            <h4 className="text-sm font-medium text-ink mb-2">使用说明</h4>
            <ul className="text-xs text-teal space-y-1">
              <li>• 全国查询：查询该名字在全国范围内的使用人数和重名率</li>
              <li>• 省份查询：查询该名字在特定省份的使用情况</li>
              <li>• 数据基于中国人口统计数据，仅供参考</li>
            </ul>
          </div>

          <div className="mt-4 p-4 bg-warm-white rounded-lg">
            <h4 className="text-sm font-medium text-ink mb-2">重名率等级说明</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2 text-xs text-teal">
              <div className="flex items-center gap-2">
                <span className="font-medium text-red-500">极高</span>
                <span>(≥5%) - 重名风险高，建议考虑其他名字</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="font-medium text-orange-500">较高</span>
                <span>(2%-5%) - 较常见，可以考虑其他名字</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="font-medium text-yellow-500">一般</span>
                <span>(0.5%-2%) - 正常水平，可以接受</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="font-medium text-green-500">较低</span>
                <span>(0.001%-0.5%) - 较少见，较为独特</span>
              </div>
              <div className="flex items-center gap-2 md:col-span-2">
                <span className="font-medium text-emerald-600">极低</span>
                <span>(&lt;0.001%) - 稀有独特，非常少见</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
