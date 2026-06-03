import { shallowMount } from '@vue/test-utils';
import PriceSuggestionForm from './PriceSuggestionForm.vue';

describe('PriceSuggestionForm', () => {
  let wrapper;

  beforeEach(() => {
    wrapper = shallowMount(PriceSuggestionForm);
  });

  it('renders the form with required fields', () => {
    expect(wrapper.find('form').exists()).toBe(true);
    expect(wrapper.find('#description').exists()).toBe(true);
    expect(wrapper.find('#requirements').exists()).toBe(true);
    expect(wrapper.find('#deliverables').exists()).toBe(true);
    expect(wrapper.find('#timeline').exists()).toBe(true);
    expect(wrapper.find('#techStack').exists()).toBe(true);
    expect(wrapper.find('#complexity').exists()).toBe(true);
    expect(wrapper.find('#constraints').exists()).toBe(true);
  });

  it('shows loading state when submitting', async () => {
    global.fetch = jest.fn(() => new Promise(() => {})); // never resolves
    wrapper.find('button').trigger('click');
    await wrapper.vm.$nextTick();
    expect(wrapper.find('button').text()).toBe('Evaluating...');
    expect(wrapper.find('button').attributes('disabled')).toBeDefined();
  });

  it('displays error on API failure', async () => {
    global.fetch = jest.fn(() =>
      Promise.resolve({
        ok: false,
        json: () => Promise.resolve({ error: 'API error' }),
      })
    );
    wrapper.find('button').trigger('click');
    await wrapper.vm.$nextTick();
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.error').text()).toBe('API error');
  });

  it('displays result and allows editing price', async () => {
    const mockResult = {
      suggestedPrice: 30000,
      suggestedPriceRange: { min: 25000, max: 35000 },
      confidenceLevel: 'high',
      breakdown: [{ category: 'Frontend', amount: 10000, description: 'UI' }],
      assumptions: ['Standard rates'],
      risks: ['Scope creep'],
    };
    global.fetch = jest.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockResult),
      })
    );
    wrapper.find('button').trigger('click');
    await wrapper.vm.$nextTick();
    await wrapper.vm.$nextTick();
    expect(wrapper.find('.result').exists()).toBe(true);
    expect(wrapper.find('#finalPrice').element.value).toBe('30000');
  });

  it('emits price-accepted event on accept', async () => {
    wrapper.vm.result = { suggestedPrice: 30000 };
    wrapper.vm.finalPrice = 28000;
    wrapper.find('.actions button').trigger('click');
    expect(wrapper.emitted('price-accepted')).toBeTruthy();
    expect(wrapper.emitted('price-accepted')[0]).toEqual([28000]);
  });
});
