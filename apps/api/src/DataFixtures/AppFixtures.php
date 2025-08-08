<?php

namespace Omed\CMS\DataFixtures;

use Doctrine\Bundle\FixturesBundle\Fixture;
use Doctrine\Persistence\ObjectManager;
use Omed\CMS\Story\DefaultArticleStory;

class AppFixtures extends Fixture
{
    public function load(ObjectManager $manager): void
    {
        // $product = new Product();
        // $manager->persist($product);
        DefaultArticleStory::load();
        $manager->flush();
    }
}
