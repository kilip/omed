<?php

namespace Omed\CMS\DataFixtures\Story;

use Omed\CMS\DataFixtures\Factory\ArticleFactory;
use Zenstruck\Foundry\Story;

final class DefaultArticleStory extends Story
{
    public function build(): void
    {
        // TODO build your story here (https://symfony.com/bundles/ZenstruckFoundryBundle/current/index.html#stories)
        ArticleFactory::createMany(100);
    }
}
