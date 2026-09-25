import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { mercenaryCardArt } from '@app/ui/tf2-art';

import { BotCard, BotForm } from './catalog';

@Component({
  selector: 'app-bot-card',
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './bot-card.html',
  styleUrl: './bot-card.scss',
})
export class BotTradingCard {
  readonly card = input.required<BotCard>();
  readonly rank = input<number>();
  readonly state = input<'available' | 'selected' | 'locked'>('available');
  readonly form = input<BotForm>('robot');
  readonly portrait = computed(() =>
    this.form() === 'human' ? mercenaryCardArt[this.card().className] : this.card().model,
  );
}
