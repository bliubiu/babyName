'use client';

import { useState } from 'react';
import { Hexagram } from '@/types';

interface HexagramDisplayProps {
  hexagram: Hexagram;
}

export default function HexagramDisplay({ hexagram }: HexagramDisplayProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };

  return (
    <div className="card mb-4 md:mb-6 lg:mb-8 animate-fadeInUp" style={{ animationDelay: '0.1s' }}>
      <div className="flex justify-between items-start mb-2 md:mb-3 lg:mb-4">
        <h2 className="font-serif text-lg md:text-xl lg:text-2xl text-ink">🔮 易经卦象</h2>
        <button
          onClick={toggleExpand}
          className="text-sm text-teal hover:text-crimson transition-colors px-3 py-1 rounded-md hover:bg-crimson/10"
        >
          {isExpanded ? '收起' : '详情'}
        </button>
      </div>
      
      <div className="flex flex-col sm:flex-row items-start gap-3 md:gap-4 lg:gap-6">
        <div className="text-3xl md:text-4xl lg:text-6xl font-serif hexagram-symbol">{hexagram.symbol}</div>
        <div className="flex-1">
          <h3 className="text-base md:text-lg lg:text-xl text-ink mb-1 md:mb-2">{hexagram.name}卦</h3>
          <p className="text-xs md:text-sm text-teal mb-1 md:mb-2">{hexagram.gua_ci}</p>
          <p className="text-sm md:text-base text-ink">{hexagram.interpretation}</p>
        </div>
      </div>

      {isExpanded && (
        <div className="mt-4 md:mt-6 space-y-4 md:space-y-6 animate-fadeIn">
          {hexagram.xiang_ci && (
            <div>
              <h4 className="font-bold text-ink mb-2 text-sm md:text-base">象辞</h4>
              <p className="text-teal text-sm md:text-base leading-relaxed">{hexagram.xiang_ci}</p>
            </div>
          )}

          {hexagram.yao_ci && hexagram.yao_ci.length > 0 && (
            <div>
              <h4 className="font-bold text-ink mb-2 text-sm md:text-base">爻辞</h4>
              <div className="space-y-2">
                {hexagram.yao_ci.map((yao, index) => (
                  <div key={index} className="flex items-start gap-2">
                    <span className="text-gold font-bold text-sm md:text-base min-w-[60px]">
                      第{index + 1}爻：
                    </span>
                    <p className="text-teal text-sm md:text-base flex-1 leading-relaxed">{yao}</p>
                  </div>
                ))}
              </div>
            </div>
          )}

          {hexagram.upper_trigram !== undefined && hexagram.lower_trigram !== undefined && (
            <div>
              <h4 className="font-bold text-ink mb-2 text-sm md:text-base">卦象分析</h4>
              <div className="grid grid-cols-2 gap-4 text-sm md:text-base">
                <div className="bg-amber-50 p-3 rounded-lg">
                  <p className="text-teal mb-1">上卦</p>
                  <p className="text-ink font-medium">第{hexagram.upper_trigram}卦</p>
                </div>
                <div className="bg-amber-50 p-3 rounded-lg">
                  <p className="text-teal mb-1">下卦</p>
                  <p className="text-ink font-medium">第{hexagram.lower_trigram}卦</p>
                </div>
              </div>
            </div>
          )}

          <div>
            <h4 className="font-bold text-ink mb-3 text-sm md:text-base">运势解读</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 md:gap-4">
              <div className="bg-blue-50 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-xl">💼</span>
                  <h5 className="font-bold text-ink text-sm md:text-base">事业</h5>
                </div>
                <p className="text-teal text-sm leading-relaxed">
                  {hexagram.name === '乾' && '天行健，君子以自强不息。事业运势强劲，宜积极进取，把握机遇。'}
                  {hexagram.name === '坤' && '地势坤，君子以厚德载物。宜稳扎稳打，积累实力。'}
                  {hexagram.name === '震' && '雷风恒，君子以立不易方。事业有变动，需谨慎应对。'}
                  {hexagram.name === '巽' && '风雷益，君子以见善则迁，有过则改。宜顺应时势，灵活变通。'}
                  {hexagram.name === '坎' && '水泽节，君子以制数度，议德行。宜节制开支，稳健发展。'}
                  {hexagram.name === '离' && '火水未济，君子以慎辨物居方。需耐心等待，时机未到。'}
                  {hexagram.name === '艮' && '山天大畜，君子以多识前言往行。宜积蓄力量，等待时机。'}
                  {hexagram.name === '兑' && '泽地萃，君子以除戎器，戒不虞。宜团结协作，共谋发展。'}
                  {!['乾', '坤', '震', '巽', '坎', '离', '艮', '兑'].includes(hexagram.name) && 
                    '根据卦象分析，事业运势平稳，宜保持现状，稳步前进。'}
                </p>
              </div>

              <div className="bg-amber-50 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-xl">💰</span>
                  <h5 className="font-bold text-ink text-sm md:text-base">财运</h5>
                </div>
                <p className="text-teal text-sm leading-relaxed">
                  {hexagram.name === '乾' && '元亨利贞，财运亨通。宜积极投资，把握商机。'}
                  {hexagram.name === '坤' && '厚德载物，财运平稳。宜稳健理财，不宜冒险。'}
                  {hexagram.name === '震' && '雷风恒，财运有波动。宜谨慎理财，控制风险。'}
                  {hexagram.name === '巽' && '风雷益，财运渐增。宜把握机会，适度投资。'}
                  {hexagram.name === '坎' && '水泽节，财运需节制。宜量入为出，避免浪费。'}
                  {hexagram.name === '离' && '火水未济，财运待时。宜耐心等待，不宜急躁。'}
                  {hexagram.name === '艮' && '山天大畜，财运积蓄。宜勤俭节约，积累财富。'}
                  {hexagram.name === '兑' && '泽地萃，财运汇聚。宜合作共赢，共享收益。'}
                  {!['乾', '坤', '震', '巽', '坎', '离', '艮', '兑'].includes(hexagram.name) && 
                    '根据卦象分析，财运平稳，宜保持理性，稳健理财。'}
                </p>
              </div>

              <div className="bg-green-50 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-xl">❤️</span>
                  <h5 className="font-bold text-ink text-sm md:text-base">爱情</h5>
                </div>
                <p className="text-teal text-sm leading-relaxed">
                  {hexagram.name === '乾' && '天行健，爱情积极。宜主动表达，把握缘分。'}
                  {hexagram.name === '坤' && '地势坤，爱情包容。宜相互理解，共同成长。'}
                  {hexagram.name === '震' && '雷风恒，爱情需恒。宜坚守承诺，珍惜感情。'}
                  {hexagram.name === '巽' && '风雷益，爱情有益。宜相互支持，共同进步。'}
                  {hexagram.name === '坎' && '水泽节，爱情需节。宜保持距离，尊重隐私。'}
                  {hexagram.name === '离' && '火水未济，爱情待成。宜耐心等待，缘分未到。'}
                  {hexagram.name === '艮' && '山天大畜，爱情积蓄。宜培养感情，循序渐进。'}
                  {hexagram.name === '兑' && '泽地萃，爱情汇聚。宜真诚相待，珍惜相遇。'}
                  {!['乾', '坤', '震', '巽', '坎', '离', '艮', '兑'].includes(hexagram.name) && 
                    '根据卦象分析，感情运势平稳，宜真诚相待，珍惜缘分。'}
                </p>
              </div>

              <div className="bg-pink-50 p-3 rounded-lg">
                <div className="flex items-center gap-2 mb-2">
                  <span className="text-xl">🏥</span>
                  <h5 className="font-bold text-ink text-sm md:text-base">健康</h5>
                </div>
                <p className="text-teal text-sm leading-relaxed">
                  {hexagram.name === '乾' && '天行健，精力充沛。宜适度运动，保持活力。'}
                  {hexagram.name === '坤' && '地势坤，身体平稳。宜注意休息，劳逸结合。'}
                  {hexagram.name === '震' && '雷风恒，需防波动。宜注意身体变化，及时就医。'}
                  {hexagram.name === '巽' && '风雷益，健康有益。宜保持良好习惯，增强体质。'}
                  {hexagram.name === '坎' && '水泽节，需防过劳。宜节制饮食，规律作息。'}
                  {hexagram.name === '离' && '火水未济，需防上火。宜清淡饮食，多喝水。'}
                  {hexagram.name === '艮' && '山天大畜，宜防积压。宜适度运动，促进循环。'}
                  {hexagram.name === '兑' && '泽地萃，宜防聚集。宜保持卫生，预防疾病。'}
                  {!['乾', '坤', '震', '巽', '坎', '离', '艮', '兑'].includes(hexagram.name) && 
                    '根据卦象分析，健康状况良好，宜保持良好习惯，定期体检。'}
                </p>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
